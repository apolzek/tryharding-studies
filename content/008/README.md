# Understanding the high cardinality problem in Prometheus

> **Author:** Vinícius Gomes Batista (*apolzek*)

Cardinality is the number of unique time series Prometheus has to keep in its head block. Every distinct combination of metric name and label values is one series, held in memory until it is flushed to disk. A single careless label (a user ID, a request ID, a UUID) can turn a handful of metrics into millions of series and OOM the instance. This experiment reproduces that behaviour with a configurable Go exporter and a local Prometheus, so the effect can be measured rather than assumed.

## Objectives

- Demonstrate the impact of high cardinality on Prometheus memory and query latency
- Show how label combinations multiply the total number of time series
- Provide a reproducible environment to explore the limits of a single Prometheus instance
- Collect practical rules for metric design that avoid cardinality explosions

## Prerequisites

- Docker 20.10+ with Compose v2
- At least 4 GB of free RAM (8 GB recommended for the extreme scenario)
- Basic familiarity with PromQL and Prometheus label model

## Project layout

```
.
├── docker-compose.yaml
├── prometheus.yml              # baseline scrape config
├── prometheus-mitigated.yml    # scrape config with metric_relabel_configs + sample_limit
├── benchmark.sh                # runs both configs, emits BENCHMARK.md
└── genmetrics/
    ├── Dockerfile
    ├── main.go
    ├── metrics.yaml
    ├── go.mod
    └── go.sum
```

`genmetrics` is a small Go exporter that reads `metrics.yaml` and emits gauges with the cartesian product of the configured labels. `prometheus.yml` scrapes it every 10 s; the Prometheus server is pinned to a 2 h retention and 5–10 min blocks so cardinality shows up quickly in the head block.

## Reproducing

### 1. Configure the scenario

Edit `genmetrics/metrics.yaml`. Each entry under `metrics:` defines a metric name, a description and a list of labels. A label is either low-cardinality (a fixed `values:` list) or high-cardinality (`high_cardinality: true` with a `count`, regex-style `pattern` and a `length`).

```yaml
metrics:
  - name: device_status
    description: "Device operational status"
    labels:
      - name: device_id
        high_cardinality: true
        count: 1000
        pattern: "[a-z0-9]"
        length: 8
      - name: region
        high_cardinality: false
        values: ["us-east", "us-west", "eu-central", "ap-south"]
```

Series for this metric: `1000 × 4 = 4,000`.

Supported patterns: `[a-z]`, `[A-Z]`, `[0-9]`, `[a-zA-Z]`, `[a-z0-9]`, `[a-zA-Z0-9]`, plus any single bracketed character class.

### 2. Start the stack

```bash
docker compose up -d --build
docker compose ps
```

Endpoints:

| Service              | URL                              |
|----------------------|----------------------------------|
| Raw metrics          | http://localhost:8090/metrics    |
| Exporter debug       | http://localhost:8090/debug      |
| Exporter info (JSON) | http://localhost:8090/info       |
| Health               | http://localhost:8090/health     |
| Prometheus UI        | http://localhost:9090            |
| TSDB status          | http://localhost:9090/tsdb-status |

`/debug` prints the exact number of series that will be exposed per metric — useful for confirming expectations before Prometheus scrapes anything.

### 3. Apply changes to `metrics.yaml`

The file is bind-mounted read-only, so `docker compose restart genmetrics` is enough:

```bash
docker compose restart genmetrics
```

### 4. Teardown

```bash
docker compose down -v
```

## Example scenarios

Low cardinality (safe):

```yaml
- name: request_count
  labels:
    - name: method
      values: ["GET", "POST", "PUT", "DELETE"]
    - name: status
      values: ["200", "404", "500"]
```

Series: `4 × 3 = 12`.

Medium cardinality (acceptable):

```yaml
- name: api_latency
  labels:
    - name: endpoint
      high_cardinality: true
      count: 50
    - name: region
      values: ["us", "eu", "asia"]
```

Series: `50 × 3 = 150`.

High cardinality (dangerous):

```yaml
- name: user_activity
  labels:
    - name: user_id
      high_cardinality: true
      count: 10000
      pattern: "[a-zA-Z0-9]"
      length: 12
    - name: action
      values: ["login", "logout", "view", "edit", "delete"]
```

Series: `10000 × 5 = 50,000`.

The default `metrics.yaml` in this repo already produces ~57,200 series (`200 + 3,000 + 54,000`), which is enough to see the effect on a laptop.

## Useful PromQL

```promql
# Total active series in the head block
prometheus_tsdb_head_series

# Prometheus resident memory
process_resident_memory_bytes{job="prometheus"}

# Series count per metric name (sorted, top 10)
topk(10, count by (__name__)({__name__=~".+"}))

# Number of distinct values per label, for a given metric
count(count by (device_id)(device_status))

# Scrape duration for the target
scrape_duration_seconds{job="genmetrics"}

# Samples ingested per second
rate(prometheus_tsdb_head_samples_appended_total[1m])

# Chunks in memory (proxy for memory pressure)
prometheus_tsdb_head_chunks
```

Prometheus also exposes `/api/v1/status/tsdb` and the UI page `/tsdb-status`, which list the top metrics, top labels and top label-value pairs by series count — the fastest way to spot an offender without writing PromQL.

## Before vs. after: measuring the fix

Talking about cardinality is cheap; measuring it is the point. `benchmark.sh` runs the stack twice — once with the baseline `prometheus.yml` and once with `prometheus-mitigated.yml` — waits 90 s for the head block to settle, and collects four signals from Prometheus itself:

- `prometheus_tsdb_head_series` — active series in memory
- `process_resident_memory_bytes{job="prometheus"}` — RSS
- `sum(rate(prometheus_tsdb_head_samples_appended_total[1m]))` — ingest throughput
- Wall-clock latency (averaged over 5 runs via `curl -w "%{time_total}"`) of three representative queries

The mitigated config applies two changes at scrape time:

```yaml
sample_limit: 5000
metric_relabel_configs:
  - source_labels: [__name__]
    regex: 'network_bytes_total|request_latency'
    action: drop
```

`network_bytes_total` carries a per-connection ID (54k series from one metric) and `request_latency` carries a per-user ID (3k series) — both are the kind of identifier that belongs in traces or logs, not in a metric label. `sample_limit` is defense in depth: if the exporter ever regresses, the scrape fails loudly instead of silently ballooning the head block.

Run it:

```bash
./benchmark.sh
# writes BENCHMARK.md and prints the table below
```

Results from a 16 GB laptop (full output in [`BENCHMARK.md`](./BENCHMARK.md)):

| Scenario  | Head series | RSS    | Samples ingested | count(all) | count(job) | topk(10) |
|-----------|------------:|-------:|-----------------:|-----------:|-----------:|---------:|
| Baseline  |     458,372 | 986 MB |        31,350/s  |    0.577s  |    0.587s  |   0.583s |
| Mitigated |       2,372 | 341 MB |           258/s  |    0.002s  |    0.001s  |   0.002s |

Relative improvement: **~193× fewer series, ~2.9× less memory, ~121× fewer samples/s, ~290× faster queries**. The query latency drop is the most striking number — it is the same PromQL against a TSDB whose postings lists are now three orders of magnitude smaller.

### Why baseline head-series (458k) is much larger than the exporter reports (57k)

`genmetrics/main.go:generateValuesFromPattern` draws *new* random strings on every tick, so the set of `device_id`, `user_id` and `connection_id` values the exporter emits grows with each scrape. After 90 s (nine scrapes) the head block has seen roughly 9 × 57k ≈ 513k distinct series, all still active. This churn is not a bug of the experiment — it is the core shape of the real-world problem: short-lived identifiers (pod IDs, request IDs, session IDs) behave exactly like this in production, and cardinality is bounded by total *unique* values seen during retention, not by peak concurrent values.

## Key findings

- Each additional high-cardinality label multiplies the total series count — the combinatorial blow-up is the whole problem, not memory per se.
- Prometheus keeps all active series in memory for the duration of the head block (2 h in this experiment). Short retention does not rescue a misbehaving exporter.
- Regex queries over high-cardinality labels are the most expensive PromQL operations; the TSDB has to intersect large postings lists.
- Healthy mid-sized instances sit between 100k and 2M active series; north of 5M usually signals a design problem; north of 10M is an incident.

## Best practices

- Never put identifiers (user ID, request ID, trace ID, UUID, email) into labels. These belong in logs or traces, not metrics.
- Avoid unbounded label values (free-form text, timestamps, full URLs). Bucketise or strip them at scrape time via `metric_relabel_configs`.
- Prefer a small, fixed label set (environment, region, status, HTTP method) and let the metric name carry the rest.
- Pre-aggregate with recording rules when a high-cardinality breakdown is only occasionally needed at query time.
- Monitor cardinality continuously: alert on `prometheus_tsdb_head_series` growth rate and inspect `/tsdb-status` before it becomes an outage.
- If high cardinality is a real product requirement (per-tenant billing, per-customer SLOs), move that data to a purpose-built backend (Mimir, Thanos, VictoriaMetrics, Cortex) rather than stretching a single Prometheus.

## References

- [Prometheus — Metric and label naming](https://prometheus.io/docs/practices/naming/)
- [Prometheus — Instrumentation best practices](https://prometheus.io/docs/practices/instrumentation/)
- [Prometheus — Storage](https://prometheus.io/docs/prometheus/latest/storage/)
- [Robust Perception — How much RAM does Prometheus need?](https://www.robustperception.io/how-much-ram-does-prometheus-2-x-need-for-cardinality-and-ingestion)
- [Grafana — What are cardinality spikes and why do they matter?](https://grafana.com/blog/2022/02/15/what-are-cardinality-spikes-and-why-do-they-matter/)
- [Announcing Prometheus 3.0](https://prometheus.io/blog/2024/11/14/prometheus-3-0/)
