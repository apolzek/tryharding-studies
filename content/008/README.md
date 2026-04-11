# Understanding the high cardinality problem in Prometheus

## Objectives

This experiment aims to:

- Demonstrate the impact of high cardinality metrics on Prometheus performance
- Analyze memory consumption and query latency when scraping metrics with varying cardinality levels
- Understand how label combinations exponentially increase the number of time series
- Provide a reproducible environment to test Prometheus behavior under high cardinality scenarios
- Establish best practices for metric design to avoid cardinality explosions

## Prerequisites

Before running this experiment, ensure you have:

- **Docker** (version 20.10 or higher)
- **Docker Compose** (version 2.0 or higher)
- At least **4GB of available RAM** (8GB recommended for high cardinality tests)
- Basic understanding of Prometheus metrics and labels
- Familiarity with YAML configuration files

## Reproducing

### Step 1: Clone or Setup the Project

Organize your project structure as follows:

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

### Step 2: Configure Metrics

Edit the `genmetrics/metrics.yaml` file to define your metrics with different cardinality levels:

```yaml
metrics:
  - name: device_status
    description: "Device operational status"
    labels:
      - name: device_id
        high_cardinality: true
        count: 1000  # Generates 1000 unique device IDs
        pattern: "[a-z0-9]"
        length: 8
      - name: region
        high_cardinality: false
        values: ["us-east", "us-west", "eu-central", "ap-south"]
```

**Cardinality Calculation:** 1000 devices × 4 regions = **4,000 time series**

### Step 3: Start the Environment

```bash
# Build and start all services
docker-compose up -d

# Check service status
docker-compose ps

# View logs
docker-compose logs -f
```

### Step 4: Access the Interfaces

- **Metrics Generator Debug Info:** http://localhost:8090/debug
- **Metrics Generator Info (JSON):** http://localhost:8090/info
- **Raw Metrics Endpoint:** http://localhost:8090/metrics
- **Prometheus UI:** http://localhost:9090

### Step 5: Monitor and Analyze

#### Check Total Series Count

In Prometheus, run the following query:

```promql
prometheus_tsdb_head_series
```

This shows the total number of active time series in memory.

#### Monitor Memory Usage

```promql
process_resident_memory_bytes{job="prometheus"}
```

#### Check Cardinality by Metric

```promql
count by (__name__) ({__name__=~".+"})
```

#### Test Query Performance

Execute queries with high cardinality labels and observe response times:

```promql
device_status{device_id=~".*"}
```

### Step 6: Experiment with Different Scenarios

Modify `metrics.yaml` to test various scenarios:

**Low Cardinality (Safe):**
```yaml
- name: request_count
  labels:
    - name: method
      values: ["GET", "POST", "PUT", "DELETE"]
    - name: status
      values: ["200", "404", "500"]
```
Total series: 4 × 3 = **12 series**

**Medium Cardinality (Acceptable):**
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

**High Cardinality (Dangerous):**
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

Restart the genmetrics service after changes:

```bash
docker-compose restart genmetrics
```

### Step 7: Stress Test

To simulate extreme cardinality:

1. Increase the `count` value in high cardinality labels to 100,000+
2. Add multiple high cardinality labels to a single metric
3. Monitor Prometheus memory consumption: `docker stats prometheus`
4. Observe query performance degradation

### Step 8: Cleanup

```bash
# Stop and remove containers
docker-compose down

# Remove volumes (deletes all data)
docker-compose down -v
```

## Results

### Expected Observations

1. **Memory Consumption:** Each time series consumes approximately 1-3KB of memory in Prometheus. With 50,000 series, expect ~150MB base memory usage.

2. **Query Latency:** Queries selecting high cardinality labels will show increased latency proportional to the number of series matched.

3. **Scrape Duration:** The `/metrics` endpoint response size grows with cardinality, increasing scrape duration.

4. **TSDB Performance:** High cardinality causes more frequent compaction cycles and increased disk I/O.

### Sample Metrics

| Scenario | Series Count | Memory Usage | Query Time (avg) |
|----------|--------------|--------------|------------------|
| Low Cardinality | ~100 | ~50MB | <10ms |
| Medium Cardinality | ~5,000 | ~100MB | 50-100ms |
| High Cardinality | ~50,000 | ~250MB | 200-500ms |
| Extreme Cardinality | ~500,000+ | 2GB+ | 1-5s |

### Key Findings

- **Exponential Growth:** Each additional high cardinality label multiplies the total series count
- **Memory Pressure:** Prometheus keeps all active series in memory, leading to OOM issues with extreme cardinality
- **Query Performance:** Regex queries on high cardinality labels are particularly expensive
- **Retention Impact:** Higher cardinality increases storage requirements proportionally

### Best Practices Learned

1. ❌ **Avoid:** Using user IDs, request IDs, or UUIDs as label values
2. ❌ **Avoid:** Unbounded label values (timestamps, random strings)
3. ✅ **Use:** Fixed, bounded label sets (environment, region, status codes)
4. ✅ **Use:** Aggregation at collection time instead of high cardinality labels
5. ✅ **Monitor:** Track cardinality growth with `prometheus_tsdb_symbol_table_size_bytes`

## References

- [Prometheus Documentation - Metric and Label Naming](https://prometheus.io/docs/practices/naming/)
- [Prometheus Best Practices - Instrumentation](https://prometheus.io/docs/practices/instrumentation/)
- [Understanding and Reducing Prometheus Memory Usage](https://www.robustperception.io/how-much-ram-does-prometheus-2-x-need-for-cardinality-and-ingestion)
- [Cardinality is Key: Understanding Prometheus Metrics](https://grafana.com/blog/2022/02/15/what-are-cardinality-spikes-and-why-do-they-matter/)
- [Prometheus TSDB Format](https://prometheus.io/docs/prometheus/latest/storage/)
- [Avoiding Cardinality Explosions](https://www.timescale.com/blog/four-types-prometheus-metrics-to-collect/)

### Additional Resources

- **Metrics Generator Source Code:** The Go application in `genmetrics/` demonstrates pattern-based label generation
- **Prometheus Configuration:** Check `prometheus.yml` for scrape configurations and retention settings
- **Docker Compose Setup:** The `docker-compose.yml` orchestrates the experiment environment

---

**Note:** This experiment is for educational purposes. Running extreme cardinality tests in production environments is strongly discouraged as it can cause service degradation or outages.