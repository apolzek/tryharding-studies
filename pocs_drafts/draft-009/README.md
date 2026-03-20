# Architecting distributed observability pipelines with OpenTelemetry and Kafka


https://grafana.com/grafana/dashboards/15983-opentelemetry-collector/
https://grafana.com/grafana/dashboards/7589-kafka-exporter-overview/

./telemetrygen metrics --otlp-endpoint localhost:4317 --otlp-insecure --duration 500s --rate 2000 --workers 200 --service "new"

telemetrygen metrics --otlp-endpoint localhost:4317 --otlp-insecure --duration 60s --rate 50 --workers 200 --metric-type Sum --otlp-metric-name test_metric --service "test-otel"



```
telemetrygen metrics \
  --otlp-endpoint localhost:4317 \
  --otlp-insecure \
  --duration 300s \
  --rate 1000 \
  --workers 50 \
  --metric-type Gauge \
  --otlp-attributes="user_id=\"user-$(shuf -i 1-100000 -n 1)\"" \
  --otlp-attributes="session_id=\"session-$(uuidgen)\"" \
  --otlp-attributes="request_id=\"req-$(uuidgen)\"" \
  --otlp-attributes="endpoint=\"/api/v1/endpoint-$(shuf -i 1-500 -n 1)\"" \
  --otlp-attributes="region=\"region-$(shuf -i 1-50 -n 1)\"" \
  --otlp-attributes="datacenter=\"dc-$(shuf -i 1-20 -n 1)\"" \
  --otlp-attributes="instance_id=\"i-$(openssl rand -hex 8)\"" \
  --otlp-attributes="version=\"v$(shuf -i 1-100 -n 1).$(shuf -i 0-99 -n 1).$(shuf -i 0-999 -n 1)\"" \
  --otlp-attributes="customer_id=\"cust-$(shuf -i 1-50000 -n 1)\"" \
  --otlp-attributes="transaction_id=\"tx-$(date +%s%N)\"" \
  --service "high-cardinality-service"
```


# Gera métricas com alta cardinalidade usando múltiplos atributos variados
telemetrygen metrics \
  --otlp-endpoint localhost:4317 \
  --otlp-insecure \
  --duration 300s \
  --rate 1000 \
  --workers 50 \
  --metric-type Gauge \
  --otlp-attributes="user_id=\"user-$(shuf -i 1-100000 -n 1)\"" \
  --otlp-attributes="session_id=\"session-$(uuidgen)\"" \
  --otlp-attributes="request_id=\"req-$(uuidgen)\"" \
  --otlp-attributes="endpoint=\"/api/v1/endpoint-$(shuf -i 1-500 -n 1)\"" \
  --otlp-attributes="region=\"region-$(shuf -i 1-50 -n 1)\"" \
  --otlp-attributes="datacenter=\"dc-$(shuf -i 1-20 -n 1)\"" \
  --otlp-attributes="instance_id=\"i-$(openssl rand -hex 8)\"" \
  --otlp-attributes="version=\"v$(shuf -i 1-100 -n 1).$(shuf -i 0-99 -n 1).$(shuf -i 0-999 -n 1)\"" \
  --otlp-attributes="customer_id=\"cust-$(shuf -i 1-50000 -n 1)\"" \
  --otlp-attributes="transaction_id=\"tx-$(date +%s%N)\"" \
  --service "high-cardinality-service"


# Gera métricas com campos contendo strings longas
telemetrygen metrics \
  --otlp-endpoint localhost:4317 \
  --otlp-insecure \
  --duration 120s \
  --rate 500 \
  --workers 20 \
  --metric-type Histogram \
  --otlp-attributes="large_field=\"$(head -c 10000 < /dev/urandom | base64 | tr -d '\n')\"" \
  --otlp-attributes="error_stack=\"$(yes 'Exception at line $(shuf -i 1-1000 -n 1): NullPointerException\n\tat com.example.service.handler.process(Handler.java:$(shuf -i 100-500 -n 1))\n' | head -n 100 | tr '\n' ' ')\"" \
  --otlp-attributes="query_string=\"SELECT * FROM very_large_table_name_with_lots_of_columns WHERE $(seq 1 50 | xargs -I {} echo 'column_{} = value_{} AND' | tr '\n' ' ') 1=1\"" \
  --otlp-attributes="json_payload=\"{$(seq 1 100 | xargs -I {} echo '\\\"field_{}\\\":\\\"$(openssl rand -base64 32)\\\",' | tr '\n' ' ' | sed 's/,$//')}\"" \
  --service "large-fields-service"


# Loop para gerar continuamente com valores dinâmicos
while true; do
  telemetrygen metrics \
    --otlp-endpoint localhost:4317 \
    --otlp-insecure \
    --duration 10s \
    --rate 2000 \
    --workers 100 \
    --metric-type Sum \
    --otlp-metric-name "extreme_cardinality_metric" \
    --otlp-attributes="timestamp=\"$(date -u +%Y-%m-%dT%H:%M:%S.%3NZ)\"" \
    --otlp-attributes="nano_id=\"$(date +%s%N)-$(openssl rand -hex 16)\"" \
    --otlp-attributes="host=\"host-$(shuf -i 1-10000 -n 1).cluster-$(shuf -i 1-100 -n 1).region-$(shuf -i 1-20 -n 1).example.com\"" \
    --otlp-attributes="path=\"/$(openssl rand -base64 32 | tr -d '/' | head -c 20)/$(openssl rand -base64 32 | tr -d '/' | head -c 20)/$(openssl rand -base64 32 | tr -d '/' | head -c 20)\"" \
    --otlp-attributes="user_agent=\"Mozilla/5.0 ($(shuf -n1 -e 'Windows' 'Mac' 'Linux' 'Android' 'iOS'); $(openssl rand -base64 200))\"" \
    --otlp-attributes="referrer=\"https://$(openssl rand -base64 50 | tr -d '/' | head -c 30).example.com/$(openssl rand -base64 100 | tr -d '/')\"" \
    --otlp-attributes="cookie=\"session=$(openssl rand -base64 500); tracking=$(openssl rand -base64 500); preferences=$(openssl rand -base64 500)\"" \
    --otlp-attributes="custom_header=\"X-Custom-Header: $(head -c 5000 < /dev/urandom | base64 | tr -d '\n')\"" \
    --service "extreme-telemetry"
  
  sleep 1
done