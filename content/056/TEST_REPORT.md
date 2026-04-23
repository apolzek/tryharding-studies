# Test report — 2026-04-22

End-to-end validation of the PoC: `docker compose up --build` from a cold state, followed by guardrail and config-push tests against the live stack.

Environment: Docker Engine 29.4.1, Docker Compose v5.1.3, Linux 6.17.0-20-generic, Go 1.24 (host).

## Summary

| Check | Result |
|---|---|
| Images build from source | PASS (after 3 fixes) |
| 8 containers start and stay UP | PASS |
| 3 agents auto-register with distinct `service.name` | PASS |
| Bootstrap configs delivered from `defaults/<service.name>.yaml` | PASS |
| 3 collectors apply their own pipelines (traces/metrics/logs) | PASS (verified via `effective_config`) |
| OTLP data flows end-to-end to `otel-sink` | PASS (all 3 signals seen) |
| Guardrail: missing `memory_limiter` → 422 | PASS |
| Guardrail: banned exporter `file` → 422 | PASS |
| Guardrail: non-allowlisted endpoint host → 422 | PASS |
| Valid config push via REST → 202 | PASS |
| Effective config reflects pushed change (`batch.timeout 2s → 10s`) | PASS |
| Rate limit: second push within 30s → 429 | PASS |
| History status flips `applying → applied` after agent ack | **FAIL** — stays on `applying` |

## Fixes applied during the run

Three blocking issues were found and corrected while the test was running. Each entry shows the symptom and the change.

### 1. Dockerfile Go version drift (opamp-server + supervisor)

**Symptom** — compose build failed at `go mod tidy`:
```
go: go.mod requires go >= 1.24.0 (running go 1.23.12; GOTOOLCHAIN=local)
```

**Cause** — running `go mod tidy` on the host (Go 1.24) had rewritten `go.mod` to `go 1.24.0`, but both Dockerfiles pinned `golang:1.23-alpine`.

**Fix** — bumped both base images to `golang:1.24-alpine`.

### 2. Supervisor image is distroless-ish — `USER root` + `RUN mkdir` fail

**Symptom**:
```
> [sup-* stage-1 3/3] RUN mkdir -p /var/lib/otelcol/supervisor /etc/otelcol-supervisor:
target sup-metrics: failed to solve: ... unable to find user root: invalid argument
```

**Cause** — `otel/opentelemetry-collector-contrib:0.117.0` has no `/etc/passwd` entry for `root` and no `/bin/sh`, so neither `USER root` nor `RUN` directives work.

**Fix** — dropped both directives from `supervisor/Dockerfile`. Mount points get auto-created by Docker. The supervisor still needs write access to the named volume, so `docker-compose.yml` now sets `user: "0:0"` on all three supervisor services.

### 3. `reports_own_logs` capability not in v0.117.0

**Symptom** — all three supervisors crash-looping:
```
failed to load config: decoding failed due to the following error(s):
'capabilities' has invalid keys: reports_own_logs
```

**Cause** — `reports_own_logs` was added to the supervisor config schema after v0.117.0.

**Fix** — removed from `supervisor-{traces,metrics,logs}.yaml`. The minimal working set at v0.117.0 is:

```yaml
capabilities:
  accepts_remote_config: true
  reports_effective_config: true
  reports_own_metrics: true
  reports_health: true
  reports_remote_config: true
```

## Evidence (live artifacts captured during the run)

### Agent registration (opamp-server logs)

```
OpAMP listening on :4320/v1/opamp
UI/API listening on :4321
agent 019db6a7a20e registered
agent 019db6a7a20e: bootstrapped from defaults/pipeline-logs.yaml
agent 019db6a7a21b registered
agent 019db6a7a21b: bootstrapped from defaults/pipeline-traces.yaml
agent 019db6a7a234 registered
agent 019db6a7a234: bootstrapped from defaults/pipeline-metrics.yaml
```

### Pipeline-specific processing confirmed (logs signal)

`otel-sink` received logs carrying attributes injected by the `attributes/tag` processor that lives only in `pipeline-logs.yaml`:

```
LogRecord #0
Body: Str(the message)
Attributes:
     -> app: Str(server)
     -> pipeline.role: Str(logs)
     -> environment: Str(poc)
```

This proves the logs pipeline was the one that processed this record — not a different collector accidentally accepting logs.

### All three signals flowing

```
ResourceSpans #0          Name: okey-dokey-0 / lets-go   (telemetrygen traces)
ResourceMetrics #0..#4    Metric #0                        (telemetrygen metrics)
LogRecord #0              pipeline.role=logs               (telemetrygen logs)
```

### Guardrails — requests and server responses

```
$ curl -X POST --data-binary @no-memory-limiter.yaml "$SRV/api/agent?id=$ID"
HTTP 422
guardrails: pipeline "traces" missing required processor "memory_limiter" (guard rail)

$ curl -X POST --data-binary @banned-file-exporter.yaml "$SRV/api/agent?id=$ID"
HTTP 422
guardrails: pipeline "traces" uses banned exporter "file" (guard rail)

$ curl -X POST --data-binary @off-allowlist-endpoint.yaml "$SRV/api/agent?id=$ID"
HTTP 422
guardrails: exporter "otlp/evil" endpoint host "evil.example.com" not allowlisted (guard rail); allowed=[otel-sink localhost 127.0.0.1]
```

### Valid push and rate limit

```
$ curl -X POST --data-binary @valid-batch-10s.yaml "$SRV/api/agent?id=$ID"
HTTP 202

$ curl -X POST --data-binary @another-valid.yaml "$SRV/api/agent?id=$ID"
HTTP 429
rate limit: wait 30s
```

### Remote change observed in effective config

Before push: `batch.timeout: 2s` (from bootstrap default).
After push:  `batch.timeout: 10s` (from push payload, reported back by the collector in its next heartbeat).

No container restart. No manual intervention. Supervisor rewrote the YAML and reloaded the collector.

## Known issue — config status stuck on `applying`

The history timeline for `pipeline-traces` after a valid push:

```
2026-04-22T19:25:37  applying  bootstrap  6817dd1fa6fa
2026-04-22T19:27:25  applying  ui         5ee0056169e9
```

Both entries remain `applying` indefinitely, even minutes after the collector has the new config running. Functionally this is cosmetic — the `effective_config` field updates correctly — but the server's history view lies about the lifecycle.

Hypotheses (not yet investigated):

1. The v0.117.0 supervisor never advances `RemoteConfigStatus.Status` past `APPLYING` for remote configs it applied successfully.
2. It does advance, but the `LastRemoteConfigHash` it echoes doesn't match the hash the server computed — so `markHistory` never finds the entry to update.
3. The enum value check in `main.go` (`protobufs.RemoteConfigStatuses_RemoteConfigStatuses_APPLIED`) is valid at compile time but doesn't match what v0.117.0 emits at runtime.

To diagnose: add a `log.Printf("rc_status: %v hash=%x", msg.RemoteConfigStatus.Status, msg.RemoteConfigStatus.LastRemoteConfigHash)` inside `onMessage` in `opamp-server/main.go` and watch what actually lands. One rebuild, 30 seconds.

## Reproducing this test

```bash
cd content/056
docker compose up --build           # first build ~2 min (cached after)

# Smoke check — all three agents connected
curl -s http://localhost:4321/api/agents | jq '.[].description["service.name"]'

# Pick an agent id and try a guardrail
AGENT=$(curl -s http://localhost:4321/api/agents | jq -r '.[] | select(.description["service.name"]=="pipeline-traces") | .instance_uid')
cat <<EOF | curl -X POST --data-binary @- "http://localhost:4321/api/agent?id=$AGENT"
receivers: {otlp: {protocols: {grpc: {endpoint: 0.0.0.0:4317}}}}
processors: {batch: {timeout: 2s}}   # no memory_limiter → should reject
exporters: {debug: {verbosity: basic}}
service:
  pipelines:
    traces: {receivers: [otlp], processors: [batch], exporters: [debug]}
EOF
# Expect: HTTP 422 with the guardrail message.

# Observe pipeline-specific processing
docker compose logs otel-sink | grep pipeline.role
```

## Files touched during the fix cycle

- `opamp-server/Dockerfile` — go 1.23 → 1.24
- `supervisor/Dockerfile` — go 1.23 → 1.24; removed `USER root`, `RUN mkdir`
- `docker-compose.yml` — added `user: "0:0"` on three supervisor services
- `supervisor/supervisor-traces.yaml`
- `supervisor/supervisor-metrics.yaml`
- `supervisor/supervisor-logs.yaml` — removed `reports_own_logs: true`

No changes needed to the Go server code (`main.go`, `guardrails.go`), the web UI, or the three default pipelines under `opamp-server/defaults/`.

## Follow-ups (out of scope for this test)

1. Fix the `applied` status tracking (see diagnosis above).
2. Port-forward Prometheus endpoint of each collector (`:8888`) to host if you want to actually scrape self-metrics.
3. Try bumping `CONTRIB_VERSION` from `v0.117.0` to something newer once you want the later capability fields back.
4. Replace the hand-written guard rails with OPA/Rego if you want policy-as-code rather than Go if-statements.
