---
title: Fluentd — coletor de logs com roteamento por tag
tags: [cncf, graduated, logs, fluentd, observability]
status: stable
---

## Fluentd (CNCF Graduated)

**O que é:** agregador/unificador de logs escrito em Ruby. Plugin-based, fala HTTP, forward, syslog, tail de arquivos, TCP/UDP, e roteia para >500 destinos (S3, ES, Loki, Kafka, BigQuery, etc).

**Quando usar (SRE day-to-day):**

- Camada de coleta unificada — apps heterogêneos mandam pelo driver `fluentd` do Docker / `fluent-bit` / HTTP, e o Fluentd roteia por tag.
- Buffering em disco — se o backend (ES/Loki/S3) cair, Fluentd segura em arquivo e re-envia.
- Enriquecimento — adicionar `env`, `region`, `pod_ip` antes de sair.
- Filtragem ANTES do destino pago (evita ingestão de DEBUG em prod).

**Quando NÃO usar:**

- Se precisa de baixíssima pegada (MB de RAM) use **Fluent Bit** (mais leve, mesmo ecossistema).
- Se já tem um coletor OpenTelemetry cobrindo traces+metrics+logs, unifique com OTel Collector.

### Cenário real

*"Minha app manda log em JSON por HTTP. Quero que ERROR vá para um storage separado (mais caro/retido por mais tempo) e o resto vá para o default, mas tudo enriquecido com env=production e hostname."*

Este POC demonstra: recebimento via HTTP (9880) e forward (24224), filtro `record_transformer` para enrichment, e roteamento condicional por tag (`app.error` vs `app.**`).

### Reproducing

```bash
cd content/044/fluentd
docker compose up -d
sleep 5
```

Envie logs:

```bash
# log normal
curl -X POST -d 'json={"msg":"user login","user_id":42}' \
  http://localhost:9880/app.info

# log de erro (rota diferente)
curl -X POST -d 'json={"msg":"payment failed","order":999}' \
  http://localhost:9880/app.error
```

Veja o roteamento:

```bash
# stdout do fluentd
docker logs cncf-fluentd --tail 20

# arquivos buffer em disco (proves que gravou)
docker exec cncf-fluentd ls -la /fluentd/log/
```

### Cleanup

```bash
docker compose down -v
```

### Tips de SRE

- **Buffering**: sempre configurar `<buffer>` com `@type file` em produção — memória perde tudo se o pod morre.
- **Backpressure**: se o destino está lento, monitore `fluentd_output_status_buffer_queue_length` via o plugin Prometheus.
- **Fluentd vs Fluent Bit**: agente na beirada (DaemonSet no k8s) = Fluent Bit. Agregador central = Fluentd.
- Em Kubernetes, prefira `fluent-bit` como DaemonSet e Fluentd como aggregator Deployment. Melhor dos dois mundos.

### References

- https://docs.fluentd.org/
- https://github.com/fluent/fluentd-kubernetes-daemonset
