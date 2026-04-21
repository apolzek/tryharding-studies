---
title: OpenTelemetry — collector unificado (traces + metrics + logs)
tags: [cncf, incubating, opentelemetry, otel, otlp, observability]
status: stable
---

## OpenTelemetry (CNCF Incubating)

**O que é:** padrão aberto para telemetria (traces + metrics + logs + profiles). Compreende:

- **API/SDK** por linguagem (Go, Java, Python, .NET, Node, Rust, etc).
- **Protocolo OTLP** (gRPC e HTTP).
- **Collector** — pipeline plugável (receiver → processor → exporter) que substitui N agentes proprietários.

**Quando usar (SRE day-to-day):**

- **Único agente** por nó/pod ao invés de datadog-agent + fluent-bit + prometheus-node-exporter + jaeger-agent.
- **Vendor-neutral**: mesma instrumentação manda para Datadog, New Relic, Honeycomb, Jaeger, Grafana, Elastic — só troca exporter.
- **Processamento no caminho**: filtering, batching, tail sampling, enrichment — no Collector, não no app.

### Cenário real

*"Minha app emite traces OTLP. Quero que passem por um collector central que faz sampling 10% e manda para Jaeger. Métricas vão para Prometheus. Logs para stdout (debug)."*

### Reproducing

```bash
cd content/044/opentelemetry
docker compose up -d
sleep 12

# Gera tráfego
curl -s http://localhost:8080/dispatch?customer=392 >/dev/null
curl -s http://localhost:8080/dispatch?customer=123 >/dev/null
```

Veja as 3 rotas:
- **Jaeger UI** (traces): http://localhost:16686 → service `frontend/driver/customer/route`
- **Prometheus-format metrics** do Collector: http://localhost:8889/metrics
- **Collector debug logs**: `docker logs cncf-otelcol --tail 50`

### Cleanup

```bash
docker compose down -v
```

### Tips de SRE

- **Tail sampling** (no Collector): mantém trace errado/lento, descarta normal. Economia massiva de storage.
- **Resource detection** processor: enriquece com `k8s.*`, `cloud.*` automaticamente.
- **Agente (DaemonSet) + Gateway (Deployment)**: agente coleta local + gateway central processa/exporta. Padrão prod.
- **Contrib vs core**: `contrib` tem 300+ componentes; `core` só o essencial. Em prod use `contrib` + pin de versão.
- **Semantic Conventions**: siga `service.name`, `http.method`, `db.system`, etc — ferramentas esperam esses nomes.

### References

- https://opentelemetry.io/docs/
- https://opentelemetry.io/docs/collector/configuration/
