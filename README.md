# tryharding-studies

Numbered collection of proofs of concept (PoCs) under `content/NNN/`. Each folder is independent and self-contained, with its own `README.md` following the template in `content/README.template`:

- **Objectives**: what the PoC tries to demonstrate
- **Prerequisites**: what needs to be installed
- **Reproducing**: exact commands to run the PoC
- **Results**: what you learn / what's observable
- **References**: useful links

Every PoC carries YAML frontmatter (`title`, `tags`, `status`). This root index is generated from that metadata by `scripts/gen_index.py`. Do not hand-edit.

## Index

- <a href="content/042">042</a> (untitled)
- <a href="content/041">041</a> Cockpit of 5 OpenTelemetry Collectors with Prometheus and a modern web UI
  `observability` `opentelemetry` `prometheus` `apexcharts` `spanmetrics` `servicegraph` `count-connector` `hostmetrics` `httpcheck` `docker-compose`
- <a href="content/040">040</a> userwatch
  `ebpf` `observability` `go` `python` `sqlite` `linux`
- 039 _missing_
- <a href="content/038">038</a> tryhard-player — local YouTube-style video player from scratch
  `frontend` `video` `nodejs` `http-range-streaming` `ffmpeg`
- <a href="content/037">037</a> k0s cluster PoC in containers (Ubuntu 24.04)
  `kubernetes` `k0s` `containers` `docker-compose` `hardening` `cni` `pod-security-admission` `local-pv` `ubuntu`
- <a href="content/036">036</a> WebSocket vs Webhook — polyglot comparison (Go, Java, .NET, Node, Python)
  `networking` `websocket` `webhook` `go` `java` `dotnet` `nodejs` `python` `docker-compose` `prometheus` `k6` `tls`
- 035 _missing_
- <a href="content/034">034</a> RedQueen — Jarvis × Resident Evil
  `ai-ml` `ollama` `fastapi` `python` `discord` `frontend`
- <a href="content/033">033</a> MCP servers for observability: Prometheus, VictoriaMetrics, and Grafana
  `ai-ml` `mcp` `observability` `prometheus` `victoriametrics` `grafana`
- <a href="content/032">032</a> GPU-accelerated face recognition + open-vocabulary object detection from a webcam
  `ai-ml` `gpu` `python` `insightface` `computer-vision`
- <a href="content/031">031</a> Frontend observability with rrweb, Faro and OpenTelemetry (React, Vue, Vanilla)
  `observability` `frontend` `opentelemetry` `faro` `rrweb` `grafana` `prometheus` `loki` `tempo` `minio` `alloy` `react` `vue` `nodejs` `fastify` `docker-compose`
- <a href="content/030">030</a> Progressive delivery on Kind — Argo CD + Argo Rollouts (canary & blue/green) with Envoy Gateway
  `kubernetes` `argo-cd` `argo-rollouts` `envoy` `ci-cd` `kind`
- 029 _missing_
- 028 _missing_
- <a href="content/027">027</a> Multi-cluster Istio mesh on Kind with Grafana observability
  `kubernetes` `istio` `service-mesh` `observability` `grafana` `kind`
- <a href="content/026">026</a> Blackbox exporter on Kind monitored by kube-prometheus-stack
  `observability` `kubernetes` `prometheus` `blackbox-exporter` `kind`
- <a href="content/025">025</a> Apache Flink Kubernetes Operator on Kind
  `data-engineering` `flink` `kubernetes` `kind`
- <a href="content/024">024</a> GitHub repository inspection bot with scheduled metadata snapshots
  `automation` `python` `github-api` `telegram-bot` `sqlite`
- <a href="content/023">023</a> eBPF hands-on lab with bpftrace in Docker
  `ebpf` `bpftrace` `linux` `observability` `docker-compose`
- <a href="content/022">022</a> Discord bot with local Whisper transcription on GPU
  `ai-ml` `discord` `whisper` `gpu` `python`
- 021 _missing_
- <a href="content/020">020</a> Polyglot distributed tracing with OpenTelemetry across Go, Java, Python and Rust
  `observability` `opentelemetry` `go` `java` `python` `rust` `jaeger` `prometheus` `docker-compose`
- <a href="content/019">019</a> Does injected latency cause queue buildup between OTel collectors?
  `observability` `opentelemetry` `toxiproxy` `networking`
- <a href="content/018">018</a> Analyzing OTLP Protocol Encodings with tcpdump
  `observability` `opentelemetry` `tcpdump` `networking` `python`
- <a href="content/017">017</a> OpenTelemetry Signal Routing with Envoy Proxy
  `observability` `opentelemetry` `envoy` `prometheus` `grafana`
- <a href="content/016">016</a> Routing OTLP Data to Kafka with Vector
  `observability` `vector` `kafka` `opentelemetry`
- <a href="content/015">015</a> Distributed Observability Pipeline with OpenTelemetry and Kafka
  `observability` `opentelemetry` `kafka` `data-engineering` `prometheus` `grafana`
- <a href="content/014">014</a> Storing OpenTelemetry Data in ClickHouse
  `observability` `opentelemetry` `clickhouse` `databases` `grafana`
- <a href="content/013">013</a> Collecting Docker Logs with Fluent Bit and Loki
  `observability` `logs` `fluentbit` `loki` `grafana` `docker-compose`
- <a href="content/012">012</a> Alerting with Prometheus, Alertmanager, Karma, and a Flask Application
  `observability` `prometheus` `alertmanager` `karma` `python` `docker-compose`
- <a href="content/011">011</a> Network Connectivity Monitoring with network_exporter on Kubernetes
  `observability` `kubernetes` `networking` `prometheus` `network-exporter` `kind`
- <a href="content/010">010</a> Frontend Observability with Grafana Faro and Grafana Alloy
  `observability` `frontend` `faro` `alloy` `grafana` `loki` `tempo` `opentelemetry`
- <a href="content/009">009</a> Quick guide to chart types and when to use them
  `observability` `grafana` `data-visualization` `python`
- <a href="content/008">008</a> Understanding the high cardinality problem in Prometheus
  `observability` `prometheus` `go`
- <a href="content/007">007</a> Processing observability data at scale with Apache Flink
  `data-engineering` `flink` `observability` `opentelemetry` `kafka` `prometheus`
- <a href="content/006">006</a> Custom OpenTelemetry Collector
  `observability` `opentelemetry` `go` `otel-collector`
- <a href="content/005">005</a> Basic log segregation with OpenTelemetry using routing connector
  `observability` `opentelemetry` `logs` `routing-connector`
- <a href="content/004">004</a> PostgreSQL Replication
  `databases` `postgres` `replication` `docker-compose`
- <a href="content/003">003</a> How to monitor PostgreSQL running in container
  `observability` `databases` `postgres` `prometheus` `grafana` `docker-compose`
- <a href="content/002">002</a> Running Prometheus, Grafana, Tempo and OpenTelemetry locally with Docker Compose
  `observability` `opentelemetry` `prometheus` `grafana` `tempo` `loki` `docker-compose`
- <a href="content/001">001</a> Exploring alternatives for load balancing and reverse proxy
  `networking` `reverse-proxy` `load-balancer` `caddy` `envoy` `haproxy` `nginx` `traefik` `docker-compose`

<!-- Regenerate with: `python3 scripts/gen_index.py` -->
<!-- find . -type f -size +10M | grep -v ".git" | sed 's|^\./||' >> .gitignore -->
