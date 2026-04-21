#!/bin/bash

# Script de teste para o OpenTelemetry Gateway

ENVOY_URL="http://localhost:4318"

echo "🧪 Testando OpenTelemetry Gateway com Envoy..."
echo ""

# Função para enviar traces
send_traces() {
    echo "📊 Enviando Traces..."
    TRACE_ID=$(uuidgen | tr -d '-' | cut -c1-32)
    SPAN_ID=$(uuidgen | tr -d '-' | cut -c1-16)
    TIMESTAMP=$(date +%s)000000000

    curl -s -X POST ${ENVOY_URL}/v1/traces \
      -H "Content-Type: application/json" \
      -d "{
        \"resourceSpans\": [{
          \"resource\": {
            \"attributes\": [{
              \"key\": \"service.name\",
              \"value\": { \"stringValue\": \"test-service\" }
            }]
          },
          \"scopeSpans\": [{
            \"spans\": [{
              \"traceId\": \"${TRACE_ID}\",
              \"spanId\": \"${SPAN_ID}\",
              \"name\": \"test-operation\",
              \"kind\": 1,
              \"startTimeUnixNano\": \"${TIMESTAMP}\",
              \"endTimeUnixNano\": \"$((TIMESTAMP + 1000000000))\"
            }]
          }]
        }]
      }"

    if [ $? -eq 0 ]; then
        echo "✅ Trace enviado com sucesso!"
    else
        echo "❌ Erro ao enviar trace"
    fi
    echo ""
}

# Função para enviar metrics
send_metrics() {
    echo "📈 Enviando Metrics..."
    TIMESTAMP=$(date +%s)000000000

    curl -s -X POST ${ENVOY_URL}/v1/metrics \
      -H "Content-Type: application/json" \
      -d "{
        \"resourceMetrics\": [{
          \"resource\": {
            \"attributes\": [{
              \"key\": \"service.name\",
              \"value\": { \"stringValue\": \"test-service\" }
            }]
          },
          \"scopeMetrics\": [{
            \"metrics\": [{
              \"name\": \"test_counter\",
              \"unit\": \"1\",
              \"sum\": {
                \"dataPoints\": [{
                  \"asInt\": \"$((RANDOM % 100))\",
                  \"timeUnixNano\": \"${TIMESTAMP}\"
                }],
                \"aggregationTemporality\": 2,
                \"isMonotonic\": true
              }
            }]
          }]
        }]
      }"

    if [ $? -eq 0 ]; then
        echo "✅ Metric enviada com sucesso!"
    else
        echo "❌ Erro ao enviar metric"
    fi
    echo ""
}

# Função para enviar logs
send_logs() {
    echo "📝 Enviando Logs..."
    TIMESTAMP=$(date +%s)000000000

    curl -s -X POST ${ENVOY_URL}/v1/logs \
      -H "Content-Type: application/json" \
      -d "{
        \"resourceLogs\": [{
          \"resource\": {
            \"attributes\": [{
              \"key\": \"service.name\",
              \"value\": { \"stringValue\": \"test-service\" }
            }]
          },
          \"scopeLogs\": [{
            \"logRecords\": [{
              \"timeUnixNano\": \"${TIMESTAMP}\",
              \"severityText\": \"INFO\",
              \"severityNumber\": 9,
              \"body\": { \"stringValue\": \"Test log message at $(date)\" }
            }]
          }]
        }]
      }"

    if [ $? -eq 0 ]; then
        echo "✅ Log enviado com sucesso!"
    else
        echo "❌ Erro ao enviar log"
    fi
    echo ""
}

# Menu
case "$1" in
    traces)
        send_traces
        ;;
    metrics)
        send_metrics
        ;;
    logs)
        send_logs
        ;;
    all)
        send_traces
        send_metrics
        send_logs
        ;;
    loop)
        echo "🔄 Enviando dados continuamente (Ctrl+C para parar)..."
        while true; do
            send_traces
            send_metrics
            send_logs
            sleep 5
        done
        ;;
    *)
        echo "Uso: $0 {traces|metrics|logs|all|loop}"
        echo ""
        echo "Exemplos:"
        echo "  $0 traces  - Envia um trace de teste"
        echo "  $0 metrics - Envia uma métrica de teste"
        echo "  $0 logs    - Envia um log de teste"
        echo "  $0 all     - Envia todos os tipos de dados"
        echo "  $0 loop    - Envia dados continuamente"
        exit 1
        ;;
esac
