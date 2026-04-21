---
title: Thanos — Prometheus em escala global (HA, long-term, deduplicação)
tags: [cncf, incubating, thanos, prometheus, metrics, long-term-storage]
status: stable
---

## Thanos (CNCF Incubating)

**O que é:** conjunto de componentes que estendem o Prometheus. Resolve 3 dores grandes:

1. **Long-term storage** — Prometheus TSDB é local e cresce. Thanos Sidecar sobe os blocos TSDB para object storage (S3/GCS/Azure Blob/MinIO) a cada 2h.
2. **Global query** — Thanos Query fan-out para N Prometheus + bucket S3 e deduplica.
3. **High availability** — rode 2 Prometheus em paralelo com `replica: A/B` no `external_labels`; Query deduplica.

**Thanos vs Cortex vs Mimir:**
- **Thanos**: push do sidecar, Prometheus nativo, mais simples.
- **Cortex**: multi-tenant pesado, mais features (rule evaluation central, alertmanager HA). Arquitetura microserviços complexa.
- **Mimir** (Grafana fork do Cortex): mesma linhagem, melhor performance reportada.

Use Thanos quando: vocês já rodam Prometheus, querem só estender com long-term e global view.

**Quando usar (SRE day-to-day):**

- Retenção >30d sem PVCs gigantes — S3 é infinito e barato.
- Global view em múltiplos clusters (dev/stg/prod).
- HA do Prometheus — 2 replicas + sidecar + dedup no Query.

### Cenário real

*"Tenho 3 clusters, cada um com 1 Prometheus de 30d retenção. Quero consultar todos em 1 Grafana, com 1 ano de histórico e zero painel quebrando quando 1 Prom reinicia."*

### Reproducing

```bash
cd content/044/thanos
docker compose up -d
sleep 8
```

- Prometheus: http://localhost:9090
- Thanos Query (UI igual Prometheus mas fala com todos os sidecars): http://localhost:10902

Na UI do Thanos Query, execute PromQL — a resposta vem de Prometheus via sidecar.

```bash
# Confirma que Query vê o sidecar como "Store"
curl -s http://localhost:10902/api/v1/stores | python3 -m json.tool | head -20
```

### Cleanup

```bash
docker compose down -v
```

### Arquitetura completa (produção)

```
Prom-A + Sidecar-A ─┐
                    ├──> Object Store (S3/GCS)
Prom-B + Sidecar-B ─┘         ↑
                              │
   Thanos Compactor ──────────┤ (faz downsampling 5m/1h + retention)
                              │
   Thanos Store Gateway ──────┤ (expõe blocos do S3 via gRPC p/ Query)
                              │
   Thanos Query ──────────────┘ (fan-out: sidecars vivos + store gateway)
```

### Tips de SRE

- **Object storage** é obrigatório em prod — configure `--objstore.config-file`. Sem isso, só faz HA, não long-term.
- **Compactor** (componente separado) — único escritor do bucket. Rode 1 replica.
- **Query Frontend** (adicional): splitting + caching de queries → dashboards rápidos.
- **Downsampling**: Compactor cria blocos 5m e 1h — consultas de 1 ano caem em 2 ordens de magnitude de latência.
- **Dedup**: Query com `--query.replica-label=replica` mescla Prom-A e Prom-B. Saída única.
- `external_labels.cluster` distingue clusters na global view.

### References

- https://thanos.io/
- https://thanos.io/tip/operating/troubleshooting.md/
