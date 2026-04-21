---
title: Jaeger — distributed tracing em docker compose
tags: [cncf, graduated, tracing, jaeger, otlp, observability]
status: stable
---

## Jaeger (CNCF Graduated)

**O que é:** sistema de distributed tracing (inspirado no Dapper). Aceita spans via OTLP/Zipkin/Jaeger-Thrift, armazena e consulta por trace-id, serviço, operação, duração.

**Quando usar (SRE day-to-day):**

- Investigar **latência de cauda**: P99 disparou, onde está o gargalo? O trace mostra qual downstream está demorando.
- Mapa de serviços (Dependencies tab) para entender blast radius antes de um deploy.
- Debug de cascata de erros — um 500 do checkout, porquê? Segue o trace-id nos logs.
- Validar tracing propagation em PRs (antes de ir pra prod).

**Quando NÃO usar:**

- Armazenamento longo prazo — Jaeger all-in-one é in-memory. Em produção use backend (Cassandra, Elasticsearch, ClickHouse via jaeger-v2).
- Como fonte de métricas — embora tenha spanmetrics, métrica é Prometheus; trace é amostrado.

### Cenário real

*"Minha API checkout virou lenta no P99 mas o P50 está ok. Os logs não mostram nada óbvio. Onde está o problema?"*

Com Jaeger você olha o trace de uma request lenta e vê: **checkout-api → payment-svc → fraud-check (1.8s)**. O gargalo está no fraud-check. Sem trace, você procuraria grep-ando logs em 4 serviços.

### Reproducing

```bash
cd content/044/jaeger
docker compose up -d
sleep 10
# gere traces clicando em http://localhost:8080 (HotROD demo oficial do Jaeger)
curl -s http://localhost:8080/dispatch?customer=392 -o /dev/null
```

Abra: **http://localhost:16686**

Na UI:
1. Service: `demo-shop`
2. Find Traces
3. Clique num trace qualquer — veja os spans, duration, tags.

Endpoints de ingestão expostos (para plugar sua app):
- OTLP gRPC: `localhost:4317`
- OTLP HTTP: `localhost:4318/v1/traces`
- Zipkin: `localhost:9411` (não exposto aqui, basta mapear)

### Gerando trace manual via curl (OTLP HTTP)

```bash
curl -X POST http://localhost:4318/v1/traces \
  -H "Content-Type: application/json" \
  -d '{
    "resourceSpans":[{
      "resource":{"attributes":[{"key":"service.name","value":{"stringValue":"manual-curl"}}]},
      "scopeSpans":[{"spans":[{
        "traceId":"5b8aa5a2d2c872e8321cf37308d69df2",
        "spanId":"051581bf3cb55c13",
        "name":"manual-span",
        "kind":2,
        "startTimeUnixNano":"'$(date +%s%N)'",
        "endTimeUnixNano":"'$(($(date +%s%N)+100000000))'"
      }]}]
    }]
  }'
```

### Cleanup

```bash
docker compose down -v
```

### Tips de SRE

- **Amostragem**: 100% de trace em prod é custo infinito. Use `parentbased_traceidratio` em 1-5% com **tail sampling** para erros.
- **Service graph** da aba Dependencies: revela dependências que você nem sabia que existiam (e que viram o blast radius quando cair).
- Em produção use **backend persistente** (Elasticsearch/Cassandra) — all-in-one é só para dev/POC.
- Propague `traceparent` em tudo (W3C) — sem isso, o trace quebra ao passar por serviço que não instrumentou.

### References

- https://www.jaegertracing.io/docs/
- https://opentelemetry.io/docs/specs/otel/trace/sdk/
