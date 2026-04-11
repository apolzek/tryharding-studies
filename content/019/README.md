# POC-019 — Latency Effects Between OTel Collectors (Toxiproxy)

Demonstrates how artificial network latency injected **between two OpenTelemetry
Collectors** affects trace delivery, metric freshness, and collector behaviour.
Latency is injected with [Toxiproxy](https://github.com/Shopify/toxiproxy) so
it can be changed at runtime without restarting any collector.

---

## Architecture

```
┌──────────────┐          ┌────────────────────────┐
│ telemetrygen │          │  Backends               │
│  traces      │          │  ┌──────────────────┐   │
│  metrics     │          │  │ Jaeger  :16686   │   │
│  logs        │          │  └──────────────────┘   │
└──────┬───────┘          │  ┌──────────────────┐   │
       │ OTLP gRPC        │  │ Prometheus :9090 │   │
       ▼ :4317            │  └──────────────────┘   │
┌──────────────┐          └──────────▲──────────────┘
│ collector-1  │  OTLP gRPC          │ OTLP gRPC
│  (edge)      │ ──────────────►  ┌──┴────────────┐
│  :4317/4318  │    Toxiproxy      │  collector-2  │
└──────────────┘    :14317         │  (aggregator) │
                        ▲          └───────────────┘
                        │
               ┌────────┴───────┐
               │   Toxiproxy    │
               │   :8474 (API)  │
               │   :14317 (proxy│
               └────────────────┘
```

**Signal flow:**

```
Generator → collector-1 → [Toxiproxy injects latency] → collector-2 → Jaeger / Prometheus / File
```

Toxiproxy sits on the gRPC connection between the two collectors and adds a
configurable `latency` toxic that delays every byte transferred in the upstream
direction.

---

## Quick Start

```bash
# Start the full stack
docker compose up -d

# Run all latency scenarios sequentially
./test-latency.sh

# Run a single scenario
./test-latency.sh baseline    # 0ms
./test-latency.sh low         # 50ms ± 10ms
./test-latency.sh medium      # 200ms ± 30ms
./test-latency.sh high        # 800ms ± 100ms
./test-latency.sh critical    # 2000ms ± 500ms
./test-latency.sh restore     # remove all toxics
```

Inspect results in:

| UI | URL |
|---|---|
| Jaeger | <http://localhost:16686> |
| Prometheus | <http://localhost:9090> |
| Toxiproxy API | <http://localhost:8474/proxies> |

---

## Latency Scenarios

### Scenario 0 — Baseline (0ms)

No toxic is applied.  The proxy passes bytes straight through.

```bash
./test-latency.sh 0
```

**Expected behaviour**

- Spans appear in Jaeger within the batch processor timeout (default **5 s**).
- Prometheus metrics for `collector-2` are fresh on every 15 s scrape.
- No errors in collector logs.

**Key observation:** The `batch` processor's `timeout: 5s` is the dominant
latency factor even at 0 ms — individual spans are held for up to 5 s before
being flushed as a batch.

---

### Scenario 1 — Low (50ms ± 10ms)

```bash
./test-latency.sh 1
```

**Expected behaviour**

- End-to-end telemetry age increases by ~50 ms on top of the batch timeout.
- Invisible to most monitoring dashboards because the batch window (5 s) dwarfs it.
- No retries; no queue growth.

**When this matters:** High-cardinality trace sampling decisions that depend on
tail-based sampling (e.g., sampling on error) are delayed by the extra network RTT.

---

### Scenario 2 — Medium (200ms ± 30ms)

```bash
./test-latency.sh 2
```

**Expected behaviour**

- Span timelines in Jaeger show a visible gap between the generator timestamp
  and the time Jaeger ingested the span.
- The exporter queue in `collector-1` begins to grow because each gRPC call now
  holds a connection open ~200 ms longer.
- Collector logs may show `sending_queue` size increasing.

**How to observe in Prometheus:**

```promql
# Queue fill level on collector-1
otelcol_exporter_queue_size{exporter="otlp"}

# Export failures (should be 0 at this latency)
otelcol_exporter_send_failed_spans_total
```

---

### Scenario 3 — High (800ms ± 100ms)

```bash
./test-latency.sh 3
```

**Expected behaviour**

- The gRPC exporter default deadline is **5 s**.  At 800 ms per call, multiple
  concurrent batches can be in-flight simultaneously, consuming the queue budget.
- Intermittent `Error sending spans` messages appear in `collector-1` logs when
  individual gRPC calls are queued behind others.
- `otelcol_exporter_send_failed_spans_total` starts incrementing.
- Prometheus scrape of `collector-2` may lag: metrics reflect data that arrived
  several batch windows late.

**How to inject via API directly (without the script):**

```bash
curl -X POST http://localhost:8474/proxies/otel-collector/toxics \
  -H "Content-Type: application/json" \
  -d '{"name":"latency","type":"latency","stream":"upstream",
       "toxicity":1.0,"attributes":{"latency":800,"jitter":100}}'
```

---

### Scenario 4 — Critical (2000ms ± 500ms)

```bash
./test-latency.sh 4
```

**Expected behaviour**

- At 2 s ± 500 ms many gRPC calls will hit the **5 s deadline** (2 s network +
  batch processing + collector overhead ≈ edge of timeout).
- The exporter retry loop in `collector-1` kicks in (`retry_on_failure` is
  enabled by default) but exhausts its budget quickly.
- **Spans are dropped** — `otelcol_exporter_send_failed_spans_total` grows
  rapidly.
- `collector-1` logs show:
  ```
  Exporting failed. Will retry the request after interval.
  Dropping data because sending_queue is full.
  ```
- Backpressure propagates: the receiver queue on `collector-1` fills and
  the telemetrygen gRPC calls start receiving `ResourceExhausted`.

**How to watch drops in real time:**

```bash
# Watch collector-1 logs
docker compose logs -f otel-collector-1 | grep -E "fail|drop|retry|queue"

# Query drop counter
curl -s 'http://localhost:9090/api/v1/query?query=otelcol_exporter_send_failed_spans_total' \
  | jq '.data.result[].value[1]'
```

---

### Scenario 5 — Restore Baseline

```bash
./test-latency.sh 5   # or: ./test-latency.sh restore
```

Removes all toxics.  The proxy continues to run so no containers need
restarting.  Within one batch cycle (≤5 s) telemetry delivery returns to normal.

---

## Changing Latency at Runtime

The Toxiproxy management API (`localhost:8474`) accepts changes at any time:

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

---

## Effect Summary

| Scenario | Latency | Jitter | Data loss | Queue pressure | Visible in Jaeger |
|---|---|---|---|---|---|
| Baseline | 0 ms | 0 ms | None | None | Yes (within 5 s) |
| Low | 50 ms | ±10 ms | None | Negligible | Yes |
| Medium | 200 ms | ±30 ms | None | Mild growth | Yes, delayed |
| High | 800 ms | ±100 ms | Possible | Significant | Intermittent gaps |
| Critical | 2000 ms | ±500 ms | Yes (drops) | Overflow / backpressure | Missing spans |

---

## Key Takeaways

1. **The `batch` processor masks small latencies.** With `timeout: 5s`, any
   network delay below a few hundred milliseconds is invisible in practice —
   the batch window dominates end-to-end telemetry age.

2. **gRPC deadline is the hard limit.** The OTLP exporter's gRPC call deadline
   determines when a batch is abandoned and retried.  The OTel Collector default
   is generous (~5–10 s) but finite.  Persistent high latency will exhaust it.

3. **Retry budget ≠ infinite.** The `retry_on_failure` configuration has a
   maximum elapsed time (default 300 s).  After that, data is **silently
   dropped** unless the pipeline is explicitly configured with a persistent
   queue.

4. **Backpressure propagates upstream.** When `collector-1`'s sending queue
   fills, it applies backpressure to the receiver — ultimately causing the
   instrumented application (or generator) to experience gRPC `ResourceExhausted`
   errors.

5. **Use a persistent queue for high-latency links.** Configure
   `sending_queue.storage` with a file-storage extension to survive transient
   network partitions without data loss.

---

## Files

```
019/
├── docker-compose.yml          — Full stack definition
├── test-latency.sh             — Automated latency scenario runner
├── config/
│   ├── otel-collector-1.yaml   — Edge collector (OTLP in → Toxiproxy out)
│   ├── otel-collector-2.yaml   — Aggregator (Toxiproxy in → Jaeger/Prometheus out)
│   └── prometheus.yml          — Prometheus scrape config
└── toxiproxy/
    ├── toxiproxy.json          — Initial proxy definition (no toxics at startup)
    └── setup.sh                — Adds 100ms default toxic at boot
```
