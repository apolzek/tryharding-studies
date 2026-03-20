# OpenTelemetry Gateway com Envoy Proxy

Setup completo de gateway de telemetria usando Envoy como proxy com cache, 3 OpenTelemetry Collectors especializados e Prometheus para monitoramento.

## 🏗️ Arquitetura

```
┌─────────────────┐
│   Aplicações    │
│   (OTLP HTTP)   │
└────────┬────────┘
         │
         ▼
┌─────────────────────────────────────┐
│          Envoy Proxy                │
│   - Cache configurado               │
│   - Load balancing                  │
│   - Health checks                   │
│   - Circuit breaker                 │
└──┬─────────┬──────────┬─────────────┘
   │         │          │
   │ /v1/    │ /v1/     │ /v1/
   │ traces  │ metrics  │ logs
   │         │          │
   ▼         ▼          ▼
┌──────┐ ┌──────┐ ┌──────┐
│Traces│ │Metrics│ │ Logs │
│ Coll │ │ Coll  │ │ Coll │
│      │ │       │ │      │
│Debug │ │Debug  │ │Debug │
└──┬───┘ └──┬────┘ └──┬───┘
   │        │         │
   └────────┴─────────┘
            │
            ▼ (métricas expostas)
      ┌──────────────┐
      │  Prometheus  │
      └──────────────┘
```

## 🚀 Como Usar

### 1. Iniciar os serviços

```bash
docker-compose up -d
```

### 2. Verificar status

```bash
docker-compose ps
docker-compose logs -f
```

### 3. Acessar interfaces

- **Envoy Admin**: http://localhost:9901
- **Prometheus**: http://localhost:9090
- **OTLP Endpoint**: http://localhost:4318

## 📊 Endpoints

| Tipo     | Path         | Destino                |
|----------|--------------|------------------------|
| Traces   | /v1/traces   | otel-collector-traces  |
| Metrics  | /v1/metrics  | otel-collector-metrics |
| Logs     | /v1/logs     | otel-collector-logs    |

## 🧪 Testando o Setup

### Enviar Traces

```bash
curl -X POST http://localhost:4318/v1/traces \
  -H "Content-Type: application/json" \
  -d '{
    "resourceSpans": [{
      "resource": {
        "attributes": [{
          "key": "service.name",
          "value": { "stringValue": "test-service" }
        }]
      },
      "scopeSpans": [{
        "spans": [{
          "traceId": "5B8EFFF798038103D269B633813FC60C",
          "spanId": "EEE19B7EC3C1B174",
          "name": "test-span",
          "kind": 1,
          "startTimeUnixNano": "1544712660000000000",
          "endTimeUnixNano": "1544712661000000000"
        }]
      }]
    }]
  }'
```

### Enviar Metrics

```bash
curl -X POST http://localhost:4318/v1/metrics \
  -H "Content-Type: application/json" \
  -d '{
    "resourceMetrics": [{
      "resource": {
        "attributes": [{
          "key": "service.name",
          "value": { "stringValue": "test-service" }
        }]
      },
      "scopeMetrics": [{
        "metrics": [{
          "name": "test_counter",
          "unit": "1",
          "sum": {
            "dataPoints": [{
              "asInt": "42",
              "timeUnixNano": "1544712660000000000"
            }],
            "aggregationTemporality": 2,
            "isMonotonic": true
          }
        }]
      }]
    }]
  }'
```

### Enviar Logs

```bash
curl -X POST http://localhost:4318/v1/logs \
  -H "Content-Type: application/json" \
  -d '{
    "resourceLogs": [{
      "resource": {
        "attributes": [{
          "key": "service.name",
          "value": { "stringValue": "test-service" }
        }]
      },
      "scopeLogs": [{
        "logRecords": [{
          "timeUnixNano": "1544712660000000000",
          "severityText": "INFO",
          "body": { "stringValue": "Test log message" }
        }]
      }]
    }]
  }'
```

## 📈 Monitoramento

### Métricas dos Collectors no Prometheus

```bash
# Ver métricas do collector de traces
http://localhost:9090/graph?g0.expr=otelcol_receiver_accepted_spans

# Ver métricas do collector de metrics
http://localhost:9090/graph?g0.expr=otelcol_receiver_accepted_metric_points

# Ver métricas do collector de logs
http://localhost:9090/graph?g0.expr=otelcol_receiver_accepted_log_records
```

### Métricas do Envoy

```bash
# Ver estatísticas do cluster
http://localhost:9901/clusters

# Ver configuração
http://localhost:9901/config_dump

# Ver métricas Prometheus
http://localhost:9901/stats/prometheus
```

## 📋 Queries Úteis no Prometheus

```promql
# Taxa de traces recebidos
rate(otelcol_receiver_accepted_spans[5m])

# Taxa de metrics recebidos
rate(otelcol_receiver_accepted_metric_points[5m])

# Taxa de logs recebidos
rate(otelcol_receiver_accepted_log_records[5m])

# Conexões ativas no Envoy
envoy_cluster_upstream_cx_active

# Taxa de requisições no Envoy
rate(envoy_cluster_upstream_rq_total[5m])
```

## 🔧 Configurações Personalizadas

### Ajustar batch size dos collectors

Edite os arquivos `otel-collector-*.yaml` e modifique:

```yaml
processors:
  batch:
    timeout: 10s
    send_batch_size: 1024  # Ajuste aqui
```

### Ajustar circuit breaker do Envoy

Edite `envoy.yaml`:

```yaml
circuit_breakers:
  thresholds:
  - priority: DEFAULT
    max_connections: 1000      # Ajuste aqui
    max_pending_requests: 1000 # Ajuste aqui
```

## 🛑 Parar os Serviços

```bash
docker-compose down
```

Para remover volumes:

```bash
docker-compose down -v
```

## 📝 Logs

Ver logs de um serviço específico:

```bash
docker-compose logs -f envoy
docker-compose logs -f otel-collector-traces
docker-compose logs -f otel-collector-metrics
docker-compose logs -f otel-collector-logs
docker-compose logs -f prometheus
```

## 🔍 Debug

Os collectors estão configurados com `debug` exporter, então você verá os dados recebidos nos logs:

```bash
docker-compose logs -f otel-collector-traces
docker-compose logs -f otel-collector-metrics
docker-compose logs -f otel-collector-logs
```

## 🎯 Features

- ✅ Envoy Proxy com cache configurado
- ✅ Load balancing entre collectors
- ✅ Health checks automáticos
- ✅ Circuit breaker para proteção
- ✅ 3 Gateways especializados (traces, metrics, logs)
- ✅ Debug exporter para visualização dos dados
- ✅ Prometheus coletando métricas de todos os componentes
- ✅ Métricas do Envoy expostas


Testing Traces (gRPC):
  telemetrygen traces --otlp-insecure --otlp-endpoint localhost:4317 --duration 10s --rate 10

  Testing Traces (HTTP):
  telemetrygen traces --otlp-insecure --otlp-http-url-path=/v1/traces --otlp-endpoint localhost:4318 --duration 10s --rate 10

  Testing Metrics (gRPC):
  telemetrygen metrics --otlp-insecure --otlp-endpoint localhost:4317 --duration 10s --rate 10

  Testing Metrics (HTTP):
  telemetrygen metrics --otlp-insecure --otlp-http-url-path=/v1/metrics --otlp-endpoint localhost:4318 --duration 10s --rate 10

  Testing Logs (gRPC):
  telemetrygen logs --otlp-insecure --otlp-endpoint localhost:4317 --duration 10s --rate 10

  Testing Logs (HTTP):
  telemetrygen logs --otlp-insecure --otlp-http-url-path=/v1/logs --otlp-endpoint localhost:4318 --duration 10s --rate 10

  If you don't have telemetrygen installed, you can run it via Docker:
  docker run --network host otel/opentelemetry-collector-contrib:0.91.0 \
    telemetrygen traces --otlp-insecure --otlp-endpoint localhost:4317 --duration 10s --rate 10