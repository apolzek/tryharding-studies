# 056 — Central OpAMP control plane for 3 OpenTelemetry Collectors

PoC that runs **3 `otelcol-contrib` instances with 3 distinct pipelines** (traces / metrics / logs) and manages their configuration centrally via **OpAMP**, with guard rails applied on every push.

## Architecture

```
                     ┌──────────────────────────────────────┐
                     │  opamp-server (custom Go, :4320/WS)  │
                     │    web UI + REST API (:4321)         │
                     │    guard rails  +  history/rollback  │
                     └──────────────────────────────────────┘
                                 ▲           ▲           ▲
                                 │ OpAMP WebSocket       │
                    ┌────────────┘           │           └──────────┐
             ┌─────────────┐         ┌─────────────┐         ┌─────────────┐
             │ sup-traces  │         │ sup-metrics │         │  sup-logs   │
             │  supervisor │         │  supervisor │         │  supervisor │
             │  + otelcol  │         │  + otelcol  │         │  + otelcol  │
             │    :4317    │         │    :4317    │         │    :4317    │
             └─────────────┘         └─────────────┘         └─────────────┘
                    ▲                       ▲                       ▲
              gen-traces              gen-metrics              gen-logs
                                                                      
                            ───────── OTLP ─────────▶  otel-sink (stdout)
```

* **opamp-server** is a ~300-line Go service built on `github.com/open-telemetry/opamp-go/server`. It keeps agents in memory, serves a web UI at :4321, and pushes remote config via WebSocket.
* **supervisor** is `opampsupervisor` from `opentelemetry-collector-contrib` — it talks OpAMP to the server, receives the full collector YAML, writes it to disk and (re)starts the collector subprocess (`/otelcol-contrib`) with the new config.
* **Three pipelines** (different initial configs seeded from `opamp-server/defaults/*.yaml`): traces, metrics, logs. Each collector is actually a general-purpose binary — it becomes "the traces collector" only because of the YAML it received.

## Communication model — ports and protocol

Only **one** network path leaves the agent host: the WebSocket back to the control plane. Everything else is loopback inside the supervisor container.

### Controller ↔ agent (what crosses the network)

| Direction | Who initiates | Port | Protocol | Path |
|---|---|---|---|---|
| supervisor → OpAMP server | **agent** | `4320/TCP` | WebSocket (`ws://` or `wss://`) | `/v1/opamp` |

- **Outbound-only from agent.** The controller never opens a connection to the agent. Firewalls only need to allow egress from supervisors to the controller.
- **One persistent WebSocket per agent.** Full-duplex: heartbeats, config pushes, status reports all multiplexed.
- **Wire format:** Protobuf — `AgentToServer` / `ServerToAgent` messages defined in `github.com/open-telemetry/opamp-go/protobufs`.
- **Alternative the spec allows** (not used here): plain HTTP polling on the same path `/v1/opamp`. `opamp-go/server` accepts both at the same endpoint; the client picks.

### REST / UI plane (controller itself)

| Direction | Port | Protocol | Used by |
|---|---|---|---|
| operator → OpAMP server | `4321/TCP` | HTTP | human UI, CI/CD, scripts |

This is the **management** port — not part of the OpAMP spec. It's how you drive the controller programmatically (see *REST API* below). Only expose it to your ops network.

### Internal to the agent host (never crosses the network)

The supervisor runs a mini OpAMP server on `127.0.0.1:<random>` and points the collector's OpAMP extension at it. You can see this in any agent's `effective_config`:

```yaml
opamp:
  server:
    ws:
      endpoint: ws://127.0.0.1:46321/v1/opamp
```

Nothing here needs to be exposed, firewalled, or reasoned about from outside the host.

### Production hardening checklist

For anything beyond this PoC:

1. **mTLS** on `:4320/v1/opamp` — `TLSConfig` on `server.StartSettings`, `server.tls:` block in each `supervisor.yaml`. Endpoint becomes `wss://controller.internal:4320/v1/opamp`.
2. **AuthN** on top of mTLS — OpAMP lets agents send arbitrary headers (`server.headers:` in `supervisor.yaml`); validate a bearer token inside the server's `OnConnecting` callback.
3. **AuthZ on `:4321`** — right now the REST API is open. Put it behind OIDC / a shared secret / a VPN before anyone but you hits it.
4. **LB/proxy tuning** — if the path goes through nginx/Envoy/ALB, make sure `Upgrade: websocket` is allowed on `/v1/opamp` and the idle timeout is long (ALB defaults kill WS at 60s).
5. **Egress allowlist** — agents only need `controller.internal:4320/TCP`. Nothing inbound.

## REST API (how to drive sync from scripts / CI)

The control plane at `:4321` is fully scriptable. A push through this API goes:

```
curl → validateConfig() [guardrails]
     → store as RemoteConfig + append history
     → conn.Send() on the agent's WebSocket  [immediate, no heartbeat wait]
```

### Endpoints

| Method | Path | Body | Purpose |
|---|---|---|---|
| `GET` | `/api/agents` | — | List all known agents with status, health, description, history, pending/effective config |
| `GET` | `/api/agent?id=<uid>` | — | Single agent detail |
| `POST` | `/api/agent?id=<uid>` | YAML (collector config) | Validate + push new config. Returns `202`, `422` (guardrail), or `429` (rate limit) |
| `POST` | `/api/validate` | YAML | Dry-run the guardrails. Returns `{ok:true}` or `{ok:false, error:"..."}` |
| `POST` | `/api/rollback?id=<uid>&hash=<hex>` | — | Re-push a previous config from history (bypasses rate limit) |

### Example — push a new config to all pipelines from CI

```bash
CTRL=http://controller.internal:4321

# 1. Dry-run validation first (no side effects)
curl -fsS -X POST "$CTRL/api/validate" --data-binary @new-traces-config.yaml
# => {"ok":true}

# 2. Resolve agent id from service.name
AGENT=$(curl -fsS "$CTRL/api/agents" \
  | jq -r '.[] | select(.description["service.name"]=="pipeline-traces") | .instance_uid')

# 3. Push. 202 = accepted and delivered to the agent.
curl -fsS -X POST "$CTRL/api/agent?id=$AGENT" --data-binary @new-traces-config.yaml
# HTTP 202

# 4. Verify the agent applied it (next heartbeat, ~1s)
curl -fsS "$CTRL/api/agent?id=$AGENT" | jq -r '.effective_config' | grep batch.timeout
```

### Example — rollback to the previous known-good config

```bash
PREV=$(curl -fsS "$CTRL/api/agent?id=$AGENT" \
  | jq -r '[.history[] | select(.status=="applied")] | .[-2].hash')
curl -fsS -X POST "$CTRL/api/rollback?id=$AGENT&hash=$PREV"
```

### Clarifying "can I push directly to OpAMP?"

You do **not** speak OpAMP directly from `curl`. OpAMP is protobuf-over-WebSocket between the control plane and the agents, and agents initiate the connection. You talk to the *control plane* via its REST API (above), and the control plane talks OpAMP on your behalf. That's the whole point of having a server — it's the thing that translates "ops intent" (REST/UI/GitOps webhook) into OpAMP messages on each agent's live WebSocket.

If you wanted to skip the REST layer you would have to write your own OpAMP client using `opamp-go/client` and connect to the agent's supervisor endpoint — possible but pointless, because the agent only trusts its configured server.

## Guard rails

Enforced in `opamp-server/guardrails.go` on every config push (UI or API):

1. **Must parse as YAML.**
2. **`service.pipelines` must be non-empty**; each pipeline needs receivers + exporters.
3. **Every pipeline must include `memory_limiter` and `batch` processors** — prevents OOMs and downstream bursts.
4. **Banned exporters** — `file` is rejected (easy to exfil data); extend `bannedExporters` in `guardrails.go`.
5. **Exporter endpoint host allowlist** — default: `otel-sink`, `localhost`, `127.0.0.1`. Extend via env `OPAMP_EXPORTER_HOSTS=host1,host2` on the server.
6. **Rate limit** — max one config push per agent per 30 s.
7. **Auto-rollback** — if the agent reports `RemoteConfigStatus_FAILED`, the server reverts to the most recent `applied` config in history.
8. **Manual rollback** — any config in the agent's history can be re-pushed from the UI.

## How to run

```bash
cd content/056
docker compose up --build
```

First build takes a few minutes (the supervisor image compiles `opampsupervisor` from source).

Open **http://localhost:4321** — within ~15s you should see three agents:
- `pipeline-traces`
- `pipeline-metrics`
- `pipeline-logs`

Each shows the pending remote config (bootstrapped from `opamp-server/defaults/pipeline-<name>.yaml`), the effective config reported by the collector, health, and full push history.

Data flow verification:

```bash
docker compose logs -f otel-sink | head -40
```

You should see traces/metrics/logs arriving from all three collectors.

## Try it — central config push

1. UI → select `pipeline-traces`.
2. Edit the YAML in the "pending remote config" textarea — e.g. change `batch.timeout: 2s` to `10s`.
3. Click **validate** (should pass).
4. Click **push to agent** — the supervisor receives the new config within seconds and restarts the collector.
5. The "effective config" panel updates on the next agent heartbeat; the history row flips from `pending` → `applying` → `applied`.

## Try it — guard rails

Paste a config without `memory_limiter` or with a banned exporter and click **push**. Server responds `422 guardrails: pipeline "traces" missing required processor "memory_limiter"`.

Paste a config that exports to `endpoint: evil.example.com:4317`. Server responds with a host-allowlist rejection.

Push twice within 30 seconds — second push returns `429 rate limit: wait 24s`.

## Try it — auto-rollback

Push a config that *validates* but makes the collector unhappy at runtime (for example, set `memory_limiter.limit_mib: 1` — far below its floor). The supervisor will report `RemoteConfigStatus_FAILED`; server logs `auto-rollback to hash <prev>` and the agent is returned to the previous working config on its next heartbeat.

## Try it — manual rollback

In the agent history table, click **rollback** next to any prior hash. That config is re-pushed (bypassing the rate limit; the rate limit only applies to POST `/api/agent`).

## Ports

| host | container | what |
|---:|---:|:---|
| 4321 | 4321 | OpAMP server web UI + REST |
| 4320 | 4320 | OpAMP WebSocket (agents connect here) |
| 4317 | 4317 | OTLP gRPC → sup-traces |
| 4327 | 4317 | OTLP gRPC → sup-metrics |
| 4337 | 4317 | OTLP gRPC → sup-logs |

## Version pinning

Both the supervisor and the collector image are pinned to `v0.117.0` (collector image tag `0.117.0`). Bump in one place:

* `docker-compose.yml` — service image tags + build args
* `supervisor/Dockerfile` — base image tag

The opamp-go server library is pinned in `opamp-server/go.mod` to `v0.23.0`.

## When to graduate past this PoC

This is intentionally minimal — one process, in-memory state, JSON file persistence, HTTP API with no auth. Things to add for anything real:

* **mTLS** between supervisors and server (`tls:` block in `supervisor.yaml`, `TLSConfig` in `server.StartSettings`).
* **AuthZ** on the REST API (shared secret or OIDC).
* **Config templates / groups** — right now each agent has an independent config. Real systems want "fleets" where pushing one template fans out to N agents.
* **Durable storage** — replace the `state.json` with Postgres/etcd if you care about HA.
* **Policy engine** — for more advanced guard rails, swap the hand-written checks for OPA/Rego or CEL.

If you want the "batteries included" version instead of hacking on this, look at **BindPlane OP** (open source OpAMP server with polished UI, fleet management, and config library) — docker-compose setup in minutes. This PoC exists to show how the protocol actually works under the hood.

## File map

```
056/
├── docker-compose.yml
├── opamp-server/
│   ├── Dockerfile
│   ├── go.mod
│   ├── main.go            # OpAMP server loop, REST API, state
│   ├── guardrails.go      # validation rules
│   ├── web/index.html     # UI (vanilla JS)
│   └── defaults/          # bootstrap configs keyed by service.name
│       ├── pipeline-traces.yaml
│       ├── pipeline-metrics.yaml
│       └── pipeline-logs.yaml
├── supervisor/
│   ├── Dockerfile         # builds opampsupervisor from contrib
│   ├── supervisor-traces.yaml
│   ├── supervisor-metrics.yaml
│   └── supervisor-logs.yaml
└── backend/
    └── otel-sink.yaml     # debug collector that logs everything
```
