## Understanding the high cardinality problem in Prometheus

### Objectives

This experiment aims to:

- Demonstrate the impact of high cardinality metrics on Prometheus performance
- Analyze memory consumption and query latency when scraping metrics with varying cardinality levels
- Understand how label combinations exponentially increase the number of time series
- Provide a reproducible environment to test Prometheus behavior under high cardinality scenarios
- Establish best practices for metric design to avoid cardinality explosions

### Prerequisites

- Docker (version 20.10 or higher)
- Docker Compose (version 2.0 or higher)
- At least 4GB of available RAM (8GB recommended for high cardinality tests)
- Basic understanding of Prometheus metrics and labels

### Reproducing

Project layout:

```
.
├── docker-compose.yml
├── prometheus.yml
├── genmetrics/
│   ├── Dockerfile
│   ├── main.go
│   ├── metrics.yaml
│   └── go.mod
```

Edit `genmetrics/metrics.yaml` to define metrics with different cardinality levels:

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

Cardinality calculation: 1000 devices × 4 regions = **4,000 time series**

Start the environment:

```bash
docker compose up -d
docker compose ps
docker compose logs -f
```

Access the interfaces:

- Metrics Generator Debug: http://localhost:8090/debug
- Metrics Generator Info: http://localhost:8090/info
- Raw Metrics Endpoint: http://localhost:8090/metrics
- Prometheus UI: http://localhost:9090

Useful Prometheus queries:

```promql
# Total active series in memory
prometheus_tsdb_head_series

# Prometheus resident memory
process_resident_memory_bytes{job="prometheus"}

# Series count per metric name
count by (__name__) ({__name__=~".+"})

# Query a high cardinality label
device_status{device_id=~".*"}
```

Experiment with different scenarios by adjusting `metrics.yaml`:

Low cardinality (safe):
```yaml
- name: request_count
  labels:
    - name: method
      values: ["GET", "POST", "PUT", "DELETE"]
    - name: status
      values: ["200", "404", "500"]
```
Total series: 4 × 3 = **12 series**

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
Total series: 50 × 3 = **150 series**

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
Total series: 10,000 × 5 = **50,000 series**

Restart after changing:

```bash
docker compose restart genmetrics
```

Cleanup:

```bash
docker compose down
docker compose down -v
```

### Results

| Scenario | Series Count | Memory Usage | Query Time (avg) |
|----------|--------------|--------------|------------------|
| Low Cardinality | ~100 | ~50MB | <10ms |
| Medium Cardinality | ~5,000 | ~100MB | 50-100ms |
| High Cardinality | ~50,000 | ~250MB | 200-500ms |
| Extreme Cardinality | ~500,000+ | 2GB+ | 1-5s |

Key findings:

- Each additional high cardinality label multiplies the total series count
- Prometheus keeps all active series in memory, leading to OOM issues at the extreme
- Regex queries on high cardinality labels are especially expensive
- Storage requirements grow proportionally with cardinality

Best practices learned:

- Avoid user IDs, request IDs, or UUIDs as label values
- Avoid unbounded label values (timestamps, random strings)
- Prefer fixed, bounded label sets (environment, region, status codes)
- Aggregate at collection time instead of relying on high cardinality labels
- Track cardinality growth with `prometheus_tsdb_symbol_table_size_bytes`

### References

```
🔗 https://prometheus.io/docs/practices/naming/
🔗 https://prometheus.io/docs/practices/instrumentation/
🔗 https://www.robustperception.io/how-much-ram-does-prometheus-2-x-need-for-cardinality-and-ingestion
🔗 https://grafana.com/blog/2022/02/15/what-are-cardinality-spikes-and-why-do-they-matter/
🔗 https://prometheus.io/docs/prometheus/latest/storage/
```
