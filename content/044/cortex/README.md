---
title: Cortex — Prometheus multi-tenant + horizontally scalable
tags: [cncf, incubating, cortex, prometheus, metrics, multi-tenant]
status: stable
---

## Cortex (CNCF Incubating)

**O que é:** Prometheus-as-a-Service: horizontally scalable, multi-tenant, long-term storage. Prometheus vanilla roda 1 processo/cluster; Cortex é um cluster de 5+ componentes (distributor, ingester, ruler, querier, store gateway, compactor, alertmanager) onde cada um escala independente.

**Cortex vs Thanos vs Mimir:**
- **Thanos** — push via sidecar. Prometheus nativo continua rodando.
- **Cortex** — push via `remote_write`. Prometheus vira "agent" stateless.
- **Mimir** (Grafana) — fork do Cortex com performance otimizada; muita gente migrou.

Use Cortex quando: SaaS interno (multi-tenant por `X-Scope-OrgID`), alta ingestão (milhões de samples/s), separação ingester vs querier.

**Quando usar (SRE day-to-day):**

- Oferecer "Prometheus" como serviço para N times com isolamento (cada time = tenant).
- Escala horizontal — adicionar ingester quando lag subir.
- Long-term em S3 + querier que lê de ingesters (recentes) + store gateway (antigo).

### Cenário real

*"Empresa com 40 times, cada um quer seu próprio 'Prometheus'. Não quero rodar 40 stacks — quero 1 backend multi-tenant."*

### Reproducing

```bash
cd content/044/cortex
docker compose up -d
sleep 10
```

- Cortex distribuidor / querier: http://localhost:9009
- Prometheus (remote_write para Cortex): http://localhost:9090

Valida pipeline:

```bash
# Prometheus está enviando métricas para Cortex
curl -s http://localhost:9009/ready  # ready

# Query via Cortex (PromQL)
curl -s 'http://localhost:9009/api/prom/api/v1/query?query=up'
```

### Multi-tenant (truque do dia-a-dia)

Em prod você adiciona `--auth.enabled=true` e cada request precisa do header:

```
X-Scope-OrgID: team-backend
```

Cada tenant tem métricas/rules/alertmanager isolados.

### Cleanup

```bash
docker compose down -v
```

### Tips de SRE

- **Ingester** é stateful — tem o bloco em memória antes de flush. Não escale para zero. Use PDBs.
- **Hashring** (consistent hashing) roteia series pela label — cuidado com rebalance em upgrade.
- **Compactor**: único writer por tenant no bucket. Roda 1 por shard.
- **Migration path**: muita empresa começa com Prometheus → Thanos (sidecar) → Cortex/Mimir (remote_write puro).
- **Grafana Mimir** é basicamente Cortex com tuning. Se começando hoje, cogite Mimir.

### References

- https://cortexmetrics.io/docs/
- https://grafana.com/oss/mimir/ (alternativa popular)
