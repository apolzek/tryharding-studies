## MCP servers for observability: Prometheus, VictoriaMetrics, and Grafana

### Objectives

Model Context Protocol (MCP) servers let an LLM talk to a backend through a narrow, typed RPC surface instead of raw HTTP. For observability the interesting question is *what each server actually exposes*: how many tools, which parts of the TSDB / UI / alerting stack they cover, how they handle authentication and transports, and how much they weigh to run. This PoC brings up five MCP servers side-by-side against real Prometheus / VictoriaMetrics / Grafana backends, runs a JSON-RPC stdio client against each, and records the concrete responses so the comparison is grounded in behavior rather than README claims.

Five servers are evaluated:

- [`VictoriaMetrics/mcp-victoriametrics`](https://github.com/VictoriaMetrics/mcp-victoriametrics) — official, Go, targets VictoriaMetrics / VM Cloud.
- [`pab1it0/prometheus-mcp-server`](https://github.com/pab1it0/prometheus-mcp-server) — Python (FastMCP), minimalist.
- [`tjhop/prometheus-mcp-server`](https://github.com/tjhop/prometheus-mcp-server) — Go, exposes the full Prometheus HTTP API plus embedded docs.
- [`giantswarm/mcp-prometheus`](https://github.com/giantswarm/mcp-prometheus) — Go, read-only, good coverage of rules/alerts/alertmanager discovery.
- [`grafana/mcp-grafana`](https://github.com/grafana/mcp-grafana) — official, Go, broadest surface (dashboards, Loki, Pyroscope, OnCall, Sift, Incident).

### Prerequisites

- Docker 24+ with Compose v2
- Python 3 (for the bundled JSON-RPC test client)
- `jq` for inspecting responses
- Ports `18081–18085` and `19091–19095` free on the host
- Go 1.26 toolchain **only if** you need to rebuild `local/mcp-prometheus` from the giantswarm repo (their image is not published to a public registry)

No API keys or cloud accounts are needed — every backend runs locally.

### Layout

```
content/033/
├── README.md
├── mcp_client.py          # minimal stdio JSON-RPC driver used by every test
├── results/               # captured tool lists + call responses per server (basic tests)
├── vm-mcp/                # VictoriaMetrics + mcp-victoriametrics
├── prom-mcp-pab1it0/      # Prometheus + pab1it0/prometheus-mcp-server
├── prom-mcp-tjhop/        # Prometheus + tjhop/prometheus-mcp-server
├── prom-mcp-giantswarm/   # Prometheus + Alertmanager + rules + giantswarm/mcp-prometheus
├── grafana-mcp/           # Prometheus + Grafana + grafana/mcp-grafana
└── otel-red/              # OTel auto-instrumentation + spanmetrics → RED via all 5 MCPs
```

Each subdirectory is self-contained: its own Compose project name, its own network, and disjoint port mappings so all five stacks can run simultaneously if you want to compare responses side-by-side.

### How the tests work

Every MCP server is driven in **stdio** mode by `mcp_client.py`, which:

1. Sends `initialize` with protocol version `2025-03-26` and captures `serverInfo`.
2. Sends `notifications/initialized`.
3. Sends `tools/list` to enumerate the exposed surface.
4. Invokes a selection of tools via `tools/call` and records the first content block.

stdio was chosen over streamable-http / SSE because it removes the session-id / `text/event-stream` handling from the test and because every server supports it, making the comparison apples-to-apples. Each Compose file *also* runs the MCP in HTTP mode on a mapped port, so the same servers are reachable from an MCP-aware client (Claude Desktop, MCP Inspector, etc.) without teardown.

### Reproducing

Pull images once:

```sh
docker pull ghcr.io/pab1it0/prometheus-mcp-server:latest
docker pull ghcr.io/tjhop/prometheus-mcp-server:latest
docker pull ghcr.io/victoriametrics/mcp-victoriametrics:latest
docker pull grafana/mcp-grafana:latest
```

`giantswarm/mcp-prometheus` is not in a public registry, so build it:

```sh
git clone --depth=1 https://github.com/giantswarm/mcp-prometheus /tmp/mcp-prometheus
cd /tmp/mcp-prometheus
cat > Dockerfile.local <<'EOF'
FROM golang:1.26.2-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-w -s" -o /out/mcp-prometheus .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /out/mcp-prometheus /usr/local/bin/mcp-prometheus
ENTRYPOINT ["/usr/local/bin/mcp-prometheus"]
CMD ["serve"]
EOF
docker build -f Dockerfile.local -t local/mcp-prometheus:latest .
```

Then, from `content/033/`, for each stack:

```sh
cd prom-mcp-pab1it0 && docker compose up -d && cd ..
python3 mcp_client.py \
  --call list_metrics '{}' \
  --call execute_query '{"query":"up"}' \
  -- docker run --rm -i --network mcp033-pab1it0_obs \
       -e PROMETHEUS_URL=http://prometheus:9090 \
       ghcr.io/pab1it0/prometheus-mcp-server:latest \
  > results/pab1it0.json
```

The per-subfolder READMEs contain the exact invocation for each server.

### Backend and port map

| Subfolder | Backend port (host) | MCP HTTP port (host) | MCP server image | Compose project / network |
| --- | --- | --- | --- | --- |
| `prom-mcp-pab1it0/`    | Prometheus `19091` | `18081` | `ghcr.io/pab1it0/prometheus-mcp-server`   | `mcp033-pab1it0` / `_obs` |
| `prom-mcp-tjhop/`      | Prometheus `19092` | `18082` | `ghcr.io/tjhop/prometheus-mcp-server`     | `mcp033-tjhop` / `_obs` |
| `prom-mcp-giantswarm/` | Prometheus `19093`, Alertmanager `19094` | `18083` | `local/mcp-prometheus` (built from source) | `mcp033-giantswarm` / `_obs` |
| `vm-mcp/`              | VictoriaMetrics `18428` | `18084` | `ghcr.io/victoriametrics/mcp-victoriametrics` | `mcp033-vm` / `_obs` |
| `grafana-mcp/`         | Grafana `19095` | `18085` | `grafana/mcp-grafana`                     | `mcp033-grafana` / `_obs` |

### Results — surface and behavior

All five servers completed `initialize → tools/list → tools/call` in stdio mode against their real backend. Tool counts are pulled verbatim from `results/*.json`, not from READMEs.

| Server | Language | Image size | `serverInfo.version` (observed) | Tool count | Transports | Auth |
| --- | --- | --- | --- | --- | --- | --- |
| `pab1it0/prometheus-mcp-server` | Python (FastMCP) | **678 MB** | `Prometheus MCP 3.1.0` | **6** | stdio, http, sse | basic-auth, bearer, mTLS |
| `tjhop/prometheus-mcp-server`   | Go | 115 MB | `prometheus-mcp-server 0.17.0` | **28** | stdio, http (+SSE) | HTTP config file (Prometheus format) |
| `giantswarm/mcp-prometheus`     | Go | **58 MB** (local build) | `mcp-prometheus dev` | **18** | stdio, sse, streamable-http | basic-auth, bearer, Mimir `X-Scope-OrgID` |
| `VictoriaMetrics/mcp-victoriametrics` | Go | 142 MB | `VictoriaMetrics v1.20.2` | **16** | stdio, http, sse | VM bearer token, VM Cloud API key |
| `grafana/mcp-grafana`           | Go | 189 MB | `mcp-grafana (devel)` | **50** | stdio, sse, streamable-http | Grafana service account token or basic-auth |

#### Tool surface by category

| Category | pab1it0 | tjhop | giantswarm | VM | grafana |
| --- | --- | --- | --- | --- | --- |
| Instant / range query | ✅ `execute_query`, `execute_range_query` | ✅ `query`, `range_query` | ✅ `execute_query`, `execute_range_query` | ✅ `query`, `query_range` | ✅ `query_prometheus`, `query_prometheus_histogram` |
| Metric / label discovery | ✅ `list_metrics`, `get_metric_metadata` | ✅ `label_names`, `label_values`, `series`, `metric_metadata` | ✅ `list_label_names`, `list_label_values`, `find_series`, `get_metric_metadata` | ✅ `metrics`, `labels`, `label_values`, `series`, `metrics_metadata` | ✅ `list_prometheus_metric_names`, `list_prometheus_label_names`, `list_prometheus_label_values`, `list_prometheus_metric_metadata` |
| Targets / scrape state | ✅ `get_targets` | ✅ `list_targets`, `targets_metadata` | ✅ `get_targets`, `get_targets_metadata` | ❌ | — (goes through Grafana datasource) |
| Build / runtime / config | ❌ | ✅ `build_info`, `runtime_info`, `config`, `flags` | ✅ `get_build_info`, `get_runtime_info`, `get_config`, `get_flags` | ❌ | ❌ |
| TSDB stats / exemplars | ❌ | ✅ `tsdb_stats`, `wal_replay_status`, `exemplar_query` | ✅ `get_tsdb_stats`, `query_exemplars` | ✅ `tsdb_status` | ❌ |
| Rules / alerts | ❌ | ✅ `list_rules`, `list_alerts`, `alertmanagers` | ✅ `get_rules`, `get_alerts`, `get_alertmanagers` | ✅ `rules`, `alerts` | ✅ `alerting_manage_rules`, `alerting_manage_routing`, `list_alert_groups`, `get_alert_group` |
| Writes / admin (dangerous) | ❌ | ✅ `delete_series`, `clean_tombstones`, `snapshot`, `reload`, `quit` (opt-in flag) | ❌ (read-only by design) | ❌ | `--disable-write` gate; dashboards/annotations/folders are write-capable |
| Health | ✅ `health_check` | ✅ `healthy`, `ready` | ✅ `check_ready` | ❌ | ❌ |
| Docs / help | ❌ | ✅ `docs_list`, `docs_read`, `docs_search` (embedded Prometheus docs) | ❌ | ✅ `documentation` (embedded VM docs) | ❌ |
| VM-specific | — | — | — | ✅ `active_queries`, `top_queries`, `metric_statistics`, `explain_query`, `prettify_query` | — |
| Dashboards / folders | — | — | — | — | ✅ `search_dashboards`, `get_dashboard_by_uid`, `get_dashboard_summary`, `get_dashboard_panel_queries`, `update_dashboard`, `create_folder`, `search_folders` |
| Annotations | — | — | — | — | ✅ `get_annotations`, `create_annotation`, `update_annotation` |
| Panel rendering | — | — | — | — | ✅ `get_panel_image` (needs grafana-image-renderer) |
| Logs (Loki) | — | — | — | — | ✅ `query_loki_logs`, `query_loki_stats`, `query_loki_patterns`, `list_loki_label_names`, `list_loki_label_values` |
| Profiles (Pyroscope) | — | — | — | — | ✅ `query_pyroscope`, `list_pyroscope_profile_types`, `list_pyroscope_label_names`, `list_pyroscope_label_values` |
| Incidents / OnCall / Sift | — | — | — | — | ✅ `create_incident`, `list_incidents`, `get_incident`, `add_activity_to_incident`, `list_oncall_*`, `get_current_oncall_users`, `*_sift_*`, `find_slow_requests`, `find_error_pattern_logs`, `get_assertions` |

#### Observed tool-call results

Every row below comes from `results/*.json` produced by `mcp_client.py` against the live stacks.

| Server | Tool invoked | Outcome (first 120 chars of content) |
| --- | --- | --- |
| pab1it0    | `list_metrics` | `{"metrics":["go_gc_cleanups_executed_cleanups_total", ...]}` — 596 series exposed |
| pab1it0    | `execute_query` / `up` | Vector with `node-exporter:9100=1` and `localhost:9090=1` |
| pab1it0    | `get_targets`   | Scrape target list returned |
| tjhop      | `query` / `up`  | Vector with both jobs up |
| tjhop      | `label_names`   | Full label-name list returned |
| tjhop      | `docs_list`     | Embedded Prometheus docs index returned |
| giantswarm | `execute_query` / `up` | Vector for prometheus / node / alertmanager |
| giantswarm | `get_alerts`    | `AlwaysFiring` alert, state=`firing`, severity=`critical` |
| giantswarm | `get_rules`     | Rule group `demo` from `/etc/prometheus/rules.yml` |
| giantswarm | `get_alertmanagers` | Active: `http://alertmanager:9093/api/v2/alerts` |
| giantswarm | `list_label_names` | 94 labels returned |
| VM         | `query` / `up`  | Vector with node + VM + vmagent up |
| VM         | `metrics`       | Full metric list from VictoriaMetrics |
| VM         | `labels`        | Label list returned |
| VM         | `tsdb_status`   | Cardinality / head-block stats |
| grafana    | `list_datasources` | `Prometheus` (uid=`prometheus`) — provisioned default |
| grafana    | `search_dashboards` | `[]` (no dashboards provisioned in this PoC) |
| grafana    | `query_prometheus` | Query reached the datasource and was executed |
| grafana    | `list_prometheus_metric_names` | `go_gc_cleanups_executed_cleanups_total`, ... |

### Results — OpenTelemetry RED (auto-instrumentation)

The tests above prove each MCP can talk to its backend. They don't say whether an LLM can actually *answer an SRE question* through one. So a second stack (`otel-red/`) adds:

- A Flask app run under **zero-code** OpenTelemetry auto-instrumentation (`opentelemetry-instrument` as the Docker `CMD`). Endpoints are designed with known latency profiles so the right answer is trivially verifiable: `/api/fast` ~5–20ms, `/api/medium` ~50–150ms, `/api/slow` ~400–1200ms, `/api/flaky` returns 500 about 35% of the time, `/api/notfound` always 404.
- A small alpine + wget loadgen producing weighted traffic.
- An `otel-collector-contrib` with the **`spanmetrics` connector**, turning the traces into classic RED metrics (`traces_spanmetrics_calls_total`, `traces_spanmetrics_duration_milliseconds_bucket`) on a `/metrics` endpoint. Both Prometheus and vmagent scrape it, so the exact same RED signals are queryable via all 5 MCPs.
- RED-based alert rules (error-rate > 5%, p95 > 300 ms) wired to Alertmanager so `get_alerts` actually has something to show.

The full walkthrough is in [`otel-red/README.md`](otel-red/README.md). The four questions asked through each MCP and the numeric answers captured in `otel-red/results/red-*.json`:

| Question | Expected | pab1it0 | tjhop | giantswarm | VM | grafana |
| --- | --- | --- | --- | --- | --- | --- |
| Request rate on `/api/fast` (rps) | ~3 | 2.64 | 2.64 | 2.64 | 3.00 | 3.18 |
| Overall error rate | ~4% | 0.034 | 0.034 | 0.034 | 0.034 | 0.046 |
| p95 latency `/api/slow` (ms) | ~1000–1200 | 1819 | 1819 | 1819 | 1806 | 1836 |
| Slowest endpoint | `/api/slow` | `/api/slow` | `/api/slow` | `/api/slow` | `/api/slow` | `/api/slow` |

All five MCPs produced the right qualitative answer. Small per-cell differences are rate-window timing (the five test runs aren't simultaneous); the labels `service_name`, `span_name`, `status_code`, `http_route` carry through OTLP → spanmetrics → scrape intact, so no MCP-specific translation is needed.

Notable detail: **Grafana's `query_prometheus` requires `endTime` even for `queryType: "instant"`** — omitting it returns `parsing end time: syntax error`. The other four MCPs just take `{"query": "..."}`.

The `DemoAppHighLatencyP95` rule fires (p95 > 300 ms because `/api/slow` dominates service-level p95) and is returned by giantswarm `get_alerts`, tjhop `list_alerts`, and VM `alerts`. `pab1it0` has no alert tool. Grafana's alert-group tools expect Grafana-managed alerting, not Prometheus-native rules, so they return empty in this setup — choose the MCP based on where the alerts actually live.

### Comparison and takeaways

**Image footprint.** `local/mcp-prometheus` (giantswarm, Alpine build) is the smallest at 58 MB; `pab1it0` is an order of magnitude larger (678 MB) because it ships a full Python + FastMCP runtime. For something embedded in a Claude / Cursor / IDE client that's paid on every startup, language choice matters.

**Coverage vs simplicity (Prometheus).** pab1it0 is deliberately narrow — 6 tools cover the "ask a question, get a number" loop and little else. tjhop is at the other extreme: 28 tools mapping almost 1-to-1 to the Prometheus HTTP API *plus* embedded docs and dangerous admin tools gated by `--dangerous.enable-tsdb-admin-tools`. giantswarm sits in the middle with 18 read-only tools and an explicit focus on rules/alerts/alertmanager (we verified: the `AlwaysFiring` alert came back cleanly via `get_alerts`, rules via `get_rules`, Alertmanager discovery via `get_alertmanagers`).

**Notifications / alerting.** Only tjhop, giantswarm, and VM (and grafana via its own alerting subsystem) expose alert state through MCP. The simplest end-to-end test — Prometheus rule → fires → visible in MCP — passed on giantswarm (see the `AlwaysFiring` entry in `results/giantswarm.json`). pab1it0 has no rule/alert tools at all.

**VictoriaMetrics-specific.** `active_queries`, `top_queries`, `metric_statistics`, `tsdb_status`, `explain_query`, and `prettify_query` have no equivalent in any Prometheus MCP — the TSDB exposes them natively and the official server surfaces them, which is the main reason to prefer it over a Prometheus MCP pointed at VictoriaMetrics' `prometheus`-compatible endpoint.

**Grafana scope.** `grafana/mcp-grafana` is the odd one out: it's not a TSDB client, it's a client for everything Grafana fronts — Prometheus *and* Loki *and* Pyroscope *and* dashboards *and* OnCall *and* Incident *and* Sift. 50 tools is a lot, but many are gated behind features that may not be installed on a given Grafana instance (e.g. OnCall, Incident, Pyroscope, the image renderer). Use `--disable-<category>` or `--enabled-tools` to trim the surface exposed to the model.

**Auth.** tjhop reuses Prometheus's own `http_config` YAML, which is convenient if you already manage TLS/basic-auth that way. giantswarm supports Mimir's `X-Scope-OrgID` out of the box (useful for multi-tenant stacks). VM supports VictoriaMetrics Cloud via `VMC_API_KEY`. pab1it0 and grafana stick to username/password or bearer.

**Transports.** All five support stdio (good for Claude Desktop / MCP Inspector). For production deployments, giantswarm and grafana default to `streamable-http`, tjhop to plain HTTP, pab1it0 auto-selects by env var, and VM via `--mode=http`.

**When to pick which:**
- Prometheus, narrow agent that only needs to answer PromQL — `pab1it0`.
- Prometheus, agent that needs the full API surface or embedded docs — `tjhop`.
- Prometheus + Alertmanager, read-only, multi-tenant safe — `giantswarm`.
- VictoriaMetrics backend — `VictoriaMetrics/mcp-victoriametrics` (the VM-specific tools matter).
- Anything that touches dashboards, logs, profiles, alerting-as-a-UI, incidents, or OnCall — `grafana/mcp-grafana`.

### References

- MCP specification — [modelcontextprotocol.io](https://modelcontextprotocol.io)
- VictoriaMetrics MCP — [github.com/VictoriaMetrics/mcp-victoriametrics](https://github.com/VictoriaMetrics/mcp-victoriametrics)
- pab1it0 Prometheus MCP — [github.com/pab1it0/prometheus-mcp-server](https://github.com/pab1it0/prometheus-mcp-server)
- tjhop Prometheus MCP — [github.com/tjhop/prometheus-mcp-server](https://github.com/tjhop/prometheus-mcp-server)
- giantswarm Prometheus MCP — [github.com/giantswarm/mcp-prometheus](https://github.com/giantswarm/mcp-prometheus)
- Grafana MCP — [github.com/grafana/mcp-grafana](https://github.com/grafana/mcp-grafana)
