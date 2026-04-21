---
title: Envoy — L7 proxy com routing, retry e health check
tags: [cncf, graduated, envoy, proxy, l7, ingress]
status: stable
---

## Envoy (CNCF Graduated)

**O que é:** proxy L7 alta performance escrito em C++. Foundation do Istio/Contour/Emissary/Gloo/AWS App Mesh. Fala HTTP/1.1, HTTP/2, HTTP/3, gRPC, TCP, UDP; suporta xDS API p/ config dinâmica.

**Quando usar (SRE day-to-day):**

- **Front proxy** sem depender de service mesh — Envoy standalone na borda.
- **gRPC-Web / HTTP to gRPC bridge** — não vive sem Envoy no meio.
- **Retry + circuit breaker** mais ricos que nginx (retry_on, retry budget, outlier detection).
- **Observabilidade** out-of-the-box: métricas Prometheus detalhadas, access log estruturado.

**Quando NÃO usar:**

- Se você só precisa de um reverse proxy HTTP simples sem tuning fino, nginx/caddy é mais ergonômico.
- Se já rodou Istio/Contour, o Envoy já está lá sem config YAML manual.

### Cenário real

*"Tenho dois backends (web e api) atrás de uma única porta. `/api/*` vai para o api com retry em 5xx. `/*` vai para o web. Quero health check ativo + access log estruturado."*

### Reproducing

```bash
cd content/044/envoy
docker compose up -d
sleep 5
```

Teste:

```bash
# rota default → web
curl -si http://localhost:8080/ | head -1

# rota /api/* → api (prefix_rewrite tira /api)
curl -si http://localhost:8080/api/ | head -1

# admin interface (stats, runtime, clusters, logging)
curl -s http://localhost:9901/stats | grep -E "^(cluster.web|cluster.api)_backend.upstream_rq_(total|5xx)" | head -10
curl -s http://localhost:9901/clusters | head -20
```

### Acessando o admin (debug de SRE)

- http://localhost:9901/ — painel admin
- `/stats` — todas as métricas
- `/clusters` — estado de cada upstream
- `/config_dump` — config efetiva (útil quando Istio gerou e você quer confirmar)
- `/runtime_modify?key=value` — flip feature flags em runtime

### Simulando backend fora do ar

```bash
docker stop cncf-envoy-api
# health_check marca UNHEALTHY em ~10s; requests p/ /api/ dão 503 até voltar
curl -si http://localhost:8080/api/ | head -3
docker start cncf-envoy-api
```

### Cleanup

```bash
docker compose down -v
```

### Tips de SRE

- **xDS em prod**: config estática (este POC) só para borda simples. Para fleet, use control plane (Istio/Gloo/custom).
- **Retry budgets**: `retry_budget` evita retry storms (retries virando ataque DDoS no upstream).
- **Outlier detection**: ejeta endpoint com 5xx bursts — sem isso, pod zombie continua recebendo tráfego.
- **Access log** → Loki/ELK. Inclui `trace_id` quando usa `tracing:`.
- **HTTP/2 downstream**: não esqueça `codec_type: AUTO` para aceitar ambos.

### References

- https://www.envoyproxy.io/docs/envoy/latest/
- https://www.envoyproxy.io/docs/envoy/latest/start/sandboxes/ (dezenas de cenários prontos)
