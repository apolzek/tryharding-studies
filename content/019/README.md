## Latency effects between OpenTelemetry Collectors with Toxiproxy

### Objectives

Demonstrate how artificial network latency injected **between two OpenTelemetry Collectors** affects trace delivery, metric freshness, and collector behaviour. Latency is injected with [Toxiproxy](https://github.com/Shopify/toxiproxy) so it can be changed at runtime without restarting any collector. The PoC explores how batch processors, gRPC deadlines, retry budgets and sending queues react to progressively worse network conditions — from 0 ms to 2 s with jitter.

### Prerequisites

- docker
- docker compose
- curl and jq (for driving the Toxiproxy API and Prometheus queries)

### Architecture

```
┌──────────────┐          ┌────────────────────────┐
│ telemetrygen │          │  Backends              │
│  traces      │          │  ┌──────────────────┐  │
│  metrics     │          │  │ Jaeger  :16686   │  │
│  logs        │          │  └──────────────────┘  │
└──────┬───────┘          │  ┌──────────────────┐  │
       │ OTLP gRPC        │  │ Prometheus :9090 │  │
       ▼ :4317            │  └──────────────────┘  │
┌──────────────┐          └──────────▲─────────────┘
│ collector-1  │  OTLP gRPC          │ OTLP gRPC
│  (edge)      │ ──────────────►  ┌──┴────────────┐
│  :4317/4318  │    Toxiproxy     │  collector-2  │
└──────────────┘    :14317        │  (aggregator) │
                        ▲         └───────────────┘
                        │
               ┌────────┴───────┐
               │   Toxiproxy    │
               │   :8474 (API)  │
               │   :14317 (prx) │
               └────────────────┘
```

Signal flow: `Generator → collector-1 → [Toxiproxy injects latency] → collector-2 → Jaeger / Prometheus / File`

### Reproducing

Start the stack:

```bash
docker compose up -d
```

Run all latency scenarios sequentially:

```bash
./test-latency.sh
```

Run a single scenario:

```bash
./test-latency.sh baseline    # 0ms
./test-latency.sh low         # 50ms  ± 10ms
./test-latency.sh medium      # 200ms ± 30ms
./test-latency.sh high        # 800ms ± 100ms
./test-latency.sh critical    # 2000ms ± 500ms
./test-latency.sh restore     # remove all toxics
```

Inspect results in:

| UI | URL |
|---|---|
| Jaeger | http://localhost:16686 |
| Prometheus | http://localhost:9090 |
| Toxiproxy API | http://localhost:8474/proxies |

Change the latency at runtime via the Toxiproxy management API:

```bash
# List current toxics
curl http://localhost:8474/proxies/otel-collector/toxics | jq

# Update existing toxic
curl -X POST http://localhost:8474/proxies/otel-collector/toxics/latency \
  -H "Content-Type: application/json" \
  -d '{"attributes":{"latency":300,"jitter":50}}'

# Remove toxic (restore baseline)
curl -X DELETE http://localhost:8474/proxies/otel-collector/toxics/latency
```

Observe drops and queue growth with:

```promql
otelcol_exporter_queue_size{exporter="otlp"}
otelcol_exporter_send_failed_spans_total
```

### Results

| Scenario | Latency | Jitter | Data loss | Queue pressure | Visible in Jaeger |
|---|---|---|---|---|---|
| Baseline | 0 ms | 0 ms | None | None | Yes (within 5 s) |
| Low | 50 ms | ±10 ms | None | Negligible | Yes |
| Medium | 200 ms | ±30 ms | None | Mild growth | Yes, delayed |
| High | 800 ms | ±100 ms | Possible | Significant | Intermittent gaps |
| Critical | 2000 ms | ±500 ms | Yes (drops) | Overflow / backpressure | Missing spans |

Key takeaways:

1. **The `batch` processor masks small latencies.** With `timeout: 5s`, any network delay below a few hundred milliseconds is invisible in practice — the batch window dominates end-to-end telemetry age.
2. **gRPC deadline is the hard limit.** The OTLP exporter's gRPC call deadline determines when a batch is abandoned and retried. Defaults are generous (~5–10 s) but finite — persistent high latency will exhaust them.
3. **Retry budget is not infinite.** `retry_on_failure` has a maximum elapsed time (default 300 s). After that, data is silently dropped unless the pipeline uses a persistent queue.
4. **Backpressure propagates upstream.** When `collector-1`'s sending queue fills, it applies backpressure to the receiver, causing the generator to experience gRPC `ResourceExhausted`.
5. **Use a persistent queue for high-latency links.** Configure `sending_queue.storage` with a file-storage extension to survive transient network partitions without data loss.

### References

```
🔗 https://github.com/Shopify/toxiproxy
🔗 https://opentelemetry.io/docs/collector/configuration/#processors
🔗 https://github.com/open-telemetry/opentelemetry-collector/blob/main/exporter/exporterhelper/README.md
🔗 https://opentelemetry.io/docs/collector/configuration/#persistent-queue
```
