---
title: Prometheus — métricas, regras e Alertmanager em docker compose
tags: [cncf, graduated, observability, prometheus, alertmanager, node-exporter]
status: stable
---

## Prometheus (CNCF Graduated)

**O que é:** sistema de monitoramento baseado em séries temporais, com coleta via pull, linguagem de query (PromQL), regras de alerta e Alertmanager para roteamento.

**Quando usar (SRE day-to-day):**

- SLOs — burn-rate alerts em cima de métricas de latência/erro.
- Capacity planning — CPU / mem / saturação dos nós antes que o oncall seja paginado.
- Fontes canônicas para dashboards de Grafana (datasource #1).
- Base para federação/long-term storage (Thanos, Cortex, Mimir).

**Quando NÃO usar:**

- Logs (use Loki/Elastic). Prometheus é métrica, não log.
- Retenção longa sem stack extra — o TSDB local cresce rápido; para >30d use Thanos/Cortex.
- Alta cardinalidade (milhares de labels dinâmicos como user_id) destrói o índice.

### Cenário real

*"Tenho uma frota de nós e quero saber quando algum passa de 80% de CPU por 2 minutos e ser notificado via webhook."*

Este POC sobe Prometheus + node-exporter + Alertmanager com uma regra `HighCPULoad` e um receiver webhook que você troca por Slack/PagerDuty em produção.

### Reproducing

```bash
cd content/044/prometheus
docker compose up -d
# espera ~10s o TSDB subir
curl -s http://localhost:9090/-/ready
```

Abra:
- Prometheus UI: http://localhost:9090
- Targets: http://localhost:9090/targets (devem estar UP)
- Alerts: http://localhost:9090/alerts
- Alertmanager: http://localhost:9093

Queries úteis para colar na UI:

```promql
# CPU % em uso (1-idle)
100 - (avg by (instance) (rate(node_cpu_seconds_total{mode="idle"}[2m])) * 100)

# Memória disponível %
node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes * 100

# Alertas ativos
ALERTS{alertstate="firing"}
```

### Forçando o alerta disparar (teste real)

```bash
# Gera carga na máquina host para ver HighCPULoad firar
docker run --rm -d --name burncpu alpine sh -c "yes > /dev/null"
# espera 2-3 min → alerta passa de PENDING p/ FIRING
docker stop burncpu
```

### Cleanup

```bash
docker compose down -v
```

### Tips de SRE

- `--web.enable-lifecycle` permite `POST /-/reload` para recarregar regras sem restart (config manager / gitops hook).
- Cardinalidade: rode `prometheus_tsdb_head_series` e `topk(10, count by (__name__)({__name__=~".+"}))` antes de achar o limite do jeito difícil.
- Antes de fazer deploy de nova regra, valide com `promtool check rules alert.rules.yml`.
- Pin da versão do image (nada de `:latest` em infra que acorda oncall).

### References

- https://prometheus.io/docs/
- https://www.robustperception.io/blog/ (dicas canônicas de PromQL)
