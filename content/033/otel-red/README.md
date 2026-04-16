## OTel auto-instrumentation + spanmetrics → RED via MCP

Same 5 MCP servers, but now against real **RED** signals produced by a Flask app running under OpenTelemetry zero-code auto-instrumentation. Traces are converted to metrics by the collector's `spanmetrics` connector, so the backend sees `traces_spanmetrics_calls_total` (Rate + Errors) and `traces_spanmetrics_duration_milliseconds_bucket` (Duration) — the canonical RED surface.

The point is to ask each MCP the kind of latency/error questions an on-call would actually ask:

- "What's the request rate per endpoint?"
- "What's the overall error rate?"
- "What is p95 latency per endpoint?"
- "Which endpoint is the slowest?"

### Layout

```
otel-red/
├── app/                     # Flask app; opentelemetry-instrument as entrypoint
│   ├── app.py               # /api/fast|medium|slow|flaky|notfound with scripted latency profiles
│   ├── Dockerfile
│   └── requirements.txt
├── loadgen/                 # alpine + wget loop generating weighted traffic
│   ├── loadgen.sh
│   └── Dockerfile
├── otel-collector-config.yaml  # OTLP in, spanmetrics connector, prometheus exporter on :8889
├── prometheus.yml            # scrapes otel-collector:8889
├── rules.yml                 # RED alert rules (error-rate > 5%, p95 > 300ms)
├── alertmanager.yml
├── vmagent.yml               # VM scrapes the same /metrics endpoint
├── datasource.yml            # Grafana provisioned datasource → Prometheus
├── docker-compose.yaml       # everything above, single stack
├── run_red_tests.sh          # drives all 5 MCPs through the 4 RED questions
└── results/red-*.json        # captured answers per MCP
```

### Data path

```
loadgen ──HTTP──▶ demo-app ──OTLP/HTTP──▶ otel-collector
                                             │
                                             ├─(spanmetrics connector → prometheus exporter :8889/metrics)
                                             │           │
                                             │           ├──▶ prometheus:9090  ──▶ pab1it0 / tjhop / giantswarm / grafana MCPs
                                             │           └──▶ vmagent ─remote_write─▶ victoriametrics:8428 ──▶ VM MCP
                                             │
                                             └─(alerts from RED rules → alertmanager → giantswarm `get_alerts`)
```

The app uses `opentelemetry-instrument` as the Docker `CMD`, which is *zero-code auto-instrumentation* — no `import opentelemetry` in `app.py`. Spans flow to the collector via OTLP/HTTP on `:4318`, the spanmetrics connector keeps them in memory and flushes aggregated `calls_total` + `duration_milliseconds_bucket` every 10 s to the prometheus exporter on `:8889`. Both Prometheus and vmagent scrape that endpoint.

### Fake latency profiles

`app.py` assigns each endpoint a known latency profile so the MCP's answer is cheap to validate:

| Endpoint | Profile | Expected p95 | Notes |
| --- | --- | --- | --- |
| `/api/fast`     | 5–20 ms    | ~20 ms      | dominant volume (5×/loop) |
| `/api/medium`   | 50–150 ms  | ~150 ms     | 2×/loop |
| `/api/slow`     | 400–1200 ms| ~1200 ms    | 1×/loop — will own p95 |
| `/api/flaky`    | 50–150 ms  | ~150 ms     | ~35% return 500 — drives error rate |
| `/api/notfound` | ~0 ms      | ~5 ms       | always 404 |

### Reproducing

```sh
# From content/033/otel-red/
docker compose up -d --build
# Give spanmetrics 60-90 s to accumulate a meaningful [1m] rate window
./run_red_tests.sh
```

Artifacts land in `results/red-<server>.json`. Each file contains `tools/list` + four `tools/call` responses with raw content from the server.

Useful backend UIs while the stack is up:

| Service | URL |
| --- | --- |
| demo-app       | http://localhost:18000/api/fast |
| OTel `/metrics`| http://localhost:18889/metrics |
| Prometheus     | http://localhost:19096 |
| Alertmanager   | http://localhost:19097 |
| VictoriaMetrics| http://localhost:18429 |
| Grafana (admin/admin) | http://localhost:19098 |

### Results — the four questions, answered by each MCP

All five servers returned consistent numbers (differences are just rate-window timing). Values below are from the captured `results/red-*.json` after the stack had run ~90 s.

**Q1. Request rate per endpoint** — `sum by (span_name) (rate(traces_spanmetrics_calls_total{service_name="demo-app"}[1m]))`

| Endpoint | pab1it0 | tjhop | giantswarm | VM | grafana |
| --- | --- | --- | --- | --- | --- |
| `GET /api/fast`     | 2.64 | 2.64 | 2.64 | 3.00 | 3.18 |
| `GET /api/medium`   | 1.05 | 1.05 | 1.05 | 1.20 | 1.27 |
| `GET /api/flaky`    | 0.53 | 0.53 | 0.53 | 0.58 | 0.65 |
| `GET /api/slow`     | 0.53 | 0.53 | 0.53 | 0.58 | 0.65 |
| `GET /api/notfound` | 0.07 | 0.07 | 0.07 | 0.07 | 0.09 |

**Q2. Overall error rate** — `sum(rate(calls_total{status_code="STATUS_CODE_ERROR"}[1m])) / sum(rate(calls_total[1m]))`

| pab1it0 | tjhop | giantswarm | VM | grafana |
| --- | --- | --- | --- | --- |
| 0.0340 | 0.0340 | 0.0340 | 0.0336 | 0.0464 |

(Expected ≈ 0.35 × rate(flaky) / total ≈ 0.04. Matches.)

**Q3. p95 latency per endpoint (ms)** — `histogram_quantile(0.95, sum by (le, span_name) (rate(duration_milliseconds_bucket[1m])))`

| Endpoint | pab1it0 | tjhop | giantswarm | VM | grafana |
| --- | --- | --- | --- | --- | --- |
| `GET /api/fast`     | 24.00   | 24.00   | 24.00   | 23.97   | 23.88   |
| `GET /api/medium`   | 231.09  | 231.09  | 231.09  | 231.38  | 234.56  |
| `GET /api/flaky`    | 233.27  | 233.27  | 233.27  | 232.50  | 230.71  |
| `GET /api/slow`     | 1818.75 | 1818.75 | 1818.75 | 1805.56 | 1836.36 |
| `GET /api/notfound` | ~4.75   | ~4.75   | ~4.75   | ~5      | ~5      |

**Q4. Slowest endpoint** — `topk(1, histogram_quantile(0.95, sum by (le, span_name) (rate(duration_milliseconds_bucket[1m]))))`

Every MCP returned `GET /api/slow` (~1.8 s), which is the right answer by construction.

### RED alerts propagated to MCP

With the stack running ~30 s the `DemoAppHighLatencyP95` rule from `rules.yml` fires (p95 > 300 ms because `/api/slow` inflates the service-level p95). Via giantswarm:

```
Active Alerts:
{Alerts:[{
  Labels:{alertname="DemoAppHighLatencyP95", service_name="demo-app", severity="warning", signal="red"}
  State:firing
  Value:~840ms
  Annotations:{summary="demo-app p95 latency above 300ms"}
}]}
```

`tjhop` exposes the same via `list_alerts`; `VictoriaMetrics/mcp-victoriametrics` via `alerts`; `grafana/mcp-grafana` via `list_alert_groups` (only if you've built Grafana-managed alerting, which this PoC does not — its alerting path is Prometheus-native, so the Prometheus MCPs are the natural consumers). `pab1it0` has no alert tool at all.

### Takeaways

- **All 5 MCPs handle OTel-sourced spanmetrics equally well** at the raw PromQL level — the label shape (`service_name`, `span_name`, `status_code`, `http_route`) is preserved through spanmetrics → Prometheus-exposition → scrape → backend, so there's no MCP-specific translation needed to answer RED questions.
- **Argument schemas are not uniform.** pab1it0 / tjhop / giantswarm / VM accept `{"query": "..."}`. Grafana's `query_prometheus` requires `datasourceUid`, `expr`, `queryType` *and* `endTime` (even for `instant`). Missing `endTime` produced `parsing end time: syntax error` — surprising for an instant query, and worth remembering when asking the Grafana MCP latency questions.
- **Rate-window drift.** Small numeric differences between servers in the tables above are not implementation differences — they're just the queries being issued a few seconds apart over a [1m] rate window where traffic is still ramping.
- **RED → alert → MCP works end-to-end.** A p95 rule defined in `rules.yml` fires in Prometheus, is picked up by Alertmanager, and surfaces through `get_alerts` on giantswarm (and `list_alerts` on tjhop). This is the minimum viable observability loop an LLM agent needs to do on-call triage: ask for metrics, ask for active alerts, correlate.
