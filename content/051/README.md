# 051 — OpenTelemetry Collector: Load Testing & Benchmarking

Benchmarking the maximum sustained throughput of the **OpenTelemetry Collector Contrib** across five resource tiers, tested separately for **OTLP/gRPC** and **OTLP/HTTP**.

## TL;DR

We stress-test a vanilla `otel/opentelemetry-collector-contrib` instance running with a neutral pipeline (`OTLP receiver → batch → nop exporter`) under Docker `cpus` / `mem_limit` caps, while `telemetrygen` drives realistic span traffic. Prometheus scrapes both the collector's internal telemetry (`:8888`) and cAdvisor, and Grafana ships with a pre-provisioned dashboard that lights up the moment you start the stack.

> Requested **v0.150.0** is not published to Docker Hub; the closest published tag in the same minor series is **v0.150.1** (same code tree, patch rebuild). `telemetrygen` does have the `v0.150.0` tag so that is what we use for the generator. Both are released under the `opentelemetry-collector-releases` pipeline.

## Layout

```
051/
├── observability/            # Prometheus + Grafana + cAdvisor (shared)
│   ├── docker-compose.yml
│   ├── prometheus/prometheus.yml
│   └── grafana/
│       ├── provisioning/     # auto-loads datasource + dashboards on startup
│       └── dashboards/otel-benchmark.json
├── otlp-grpc/                # Collector exposing ONLY OTLP/gRPC (:4317)
│   ├── docker-compose.yml
│   └── collector-config.yaml
├── otlp-http/                # Collector exposing ONLY OTLP/HTTP (:4318)
│   ├── docker-compose.yml
│   └── collector-config.yaml
├── scripts/
│   ├── run-matrix.sh         # orchestrator: 5 tiers × 2 protocols
│   └── sweep.sh              # single-tier stepped sweep + Prom queries
└── results/                  # CSVs produced by the sweep
```

## Methodology

Each test loop is one **stepped sweep** for a given `(protocol, cpu_limit, mem_limit)`:

1. Destroy & recreate the collector with the requested `cpus` / `mem_limit` cgroup caps.
2. Wait until the `health_check` extension reports available on `:13133`.
3. For each rate in a growing ladder, run `telemetrygen traces` for 30s against the isolated endpoint.
    * Each span carries 8 realistic resource/attribute tags (`service.name`, `http.method`, …) so the serializer work is not trivial.
    * When a target rate exceeds what one generator container can drive (~150k spans/s empirically), the script fans out to 2/4/6/10/16 parallel `telemetrygen` containers.
4. Query Prometheus over the steady-state window for:
    * `otelcol_receiver_accepted_spans_total` (throughput actually ingested by the collector)
    * `otelcol_receiver_refused_spans_total` (backpressure signal)
    * `otelcol_process_cpu_seconds_total` (CPU cores used by the collector process)
    * `otelcol_process_memory_rss_bytes` (process RSS inside the cgroup)
5. Early-stop the sweep when **any** of the following holds:
    * `received_rate < 85% of target` (saturation — the collector can't keep up)
    * `refused_rate ≥ 1% of target` (receiver back-pressuring)
    * Container is no longer `Running` (OOM-killed)

The highest step that cleared all three gates is reported as **best sustained rate**.

### Why "neutral pipeline"?

The benchmark stresses the receive → batch → handoff path, the part of the collector most deployments share. The sink is the built-in `nop` exporter, which accepts and drops every batch with no I/O and no serialization cost. This decouples the collector's processing throughput from whatever backend you would ship to in production — the ceilings here are an upper bound; adding a real exporter (OTLP/Kafka/Loki/…) will be lower.

### Why multiple `telemetrygen` containers?

A single `telemetrygen` container with 50 workers tops out at ~150k spans/s on this host due to HTTP/2 stream concurrency and goroutine scheduling on the client side. To keep the **client** from being the bottleneck we fan out horizontally. Each generator runs in the same `otelbench` Docker network and hits the same collector endpoint, so from the collector's perspective it is still one sustained request stream — just served by multiple TCP connections.

## Running it yourself

```bash
# 0. Make sure ports 3001, 4317, 4318, 8081, 8888, 8889, 9091, 13133, 13134 are free
docker network create otelbench

# 1. Start observability (Prometheus + Grafana + cAdvisor)
docker compose -f observability/docker-compose.yml up -d

# 2. Run the full matrix (≈60 min)
./scripts/run-matrix.sh

# 2b. OR run one cell at a time
./scripts/sweep.sh grpc 2 4g traces     # protocol, cpus, mem, signal

# 3. Open the dashboards
xdg-open http://localhost:3001     # Grafana (anonymous admin)
xdg-open http://localhost:9091     # Prometheus
```

Environment variables accepted by `sweep.sh`:

| Var              | Default | Purpose                                       |
|------------------|---------|-----------------------------------------------|
| `STEP_DURATION`  | `30`    | Seconds of traffic per rate step              |
| `WORKERS_PER_GEN`| `50`    | Goroutines per generator container            |
| `PROM`           | `http://localhost:9091` | Prometheus URL the sweep queries |

## Resource matrix tested

| Tier | CPU limit | Memory limit |
|------|-----------|--------------|
| T1   | 0.5 vCPU  | 256 MiB      |
| T2   | 1 vCPU    | 512 MiB      |
| T3   | 1 vCPU    | 1 GiB        |
| T4   | 2 vCPU    | 2 GiB        |
| T5   | 2 vCPU    | 4 GiB        |

## Results

<!-- RESULTS_START -->
> Populated automatically by `scripts/run-matrix.sh`. See `results/best.csv` for machine-readable form and `results/sweep_*.csv` for step-by-step data.
<!-- RESULTS_END -->

### Host used for the runs

<!-- HOST_START -->
_Populated after the matrix finishes._
<!-- HOST_END -->

## Versions

| Component      | Version                                                               |
|----------------|-----------------------------------------------------------------------|
| Collector      | `otel/opentelemetry-collector-contrib:0.150.1`                        |
| Generator      | `ghcr.io/open-telemetry/opentelemetry-collector-contrib/telemetrygen:v0.150.0` |
| Prometheus     | `prom/prometheus:v3.4.0`                                              |
| Grafana        | `grafana/grafana:11.6.0` (dashboards + datasource provisioned)        |
| cAdvisor       | `gcr.io/cadvisor/cadvisor:v0.52.0`                                    |

## Known caveats

- **cAdvisor on Docker 29 + overlayfs**: v0.52.0 fails to resolve the read-write layer ID for each container (`Failed to create existing container: … identify the read-write layer ID …`), so it cannot emit `name=<container>` labels for Docker scopes. The dashboards therefore rely on the collector's own `otelcol_process_*` metrics for per-process CPU/RSS — which turned out to be more precise than cgroup sampling anyway. cAdvisor still serves the host-level and cgroup-level panels.
- **Span shape**: tiny attributed spans; real-world spans with links, larger payloads, sampled trace context, or deep nesting will reduce the numbers below. Treat the figures here as upper bounds for the given resource envelope.
- **`nop` exporter**: no serialization, no network I/O. A real exporter will add per-batch CPU (encoding) plus serialization memory; expect a 2–5× drop for OTLP/HTTP to a remote sink, more for batching-heavy backends like Kafka.
- **Batch processor**: `timeout=200ms`, `send_batch_size=8192`, `send_batch_max_size=10000` — these are the collector defaults tuned slightly for throughput rather than latency.
