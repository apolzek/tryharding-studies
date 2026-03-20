## Building a custom Opentelemetry Collector with custom processor

### Objectives

The goal of this PoC is to create a customized version of the OpenTelemetry Collector Contrib. Essentially, I will use some of the receivers and exporters from the project, but we will also implement two custom processors. The first, called sleepprocessor, introduces a configurable delay (in milliseconds) into the processing pipeline. The second, called alertprocessor, triggers a webhook alert based on specific conditions, such as service.name or custom attributes (either key or value). In the end, a binary will be generated, and using a config.yaml file, it will be possible to test both the collector and the custom processors.

### Prerequisites

- curl
- ocb
- docker
- docker compose

### Reproducing

Download OpenTelemetry Collector Builder (ocb)
```
curl --proto '=https' --tlsv1.2 -fL -o ocb \
https://github.com/open-telemetry/opentelemetry-collector-releases/releases/download/cmd%2Fbuilder%2Fv0.135.0/ocb_0.135.0_linux_amd64
chmod +x ocb
sudo mv ocb /usr/local/bin
```

Create builder-config.yaml
```yaml
dist:
  name: custom-collector
  description: "Custom collector"
  output_path: ./dist
  module: github.com/apolzek/custom-collector
  version: "0.1.0"
  otelcol_version: "0.135.0"

receivers:
  - gomod: go.opentelemetry.io/collector/receiver/otlpreceiver v0.135.0
  - gomod: github.com/open-telemetry/opentelemetry-collector-contrib/receiver/prometheusreceiver v0.135.0

processors:
  - gomod: github.com/apolzek/sherlock-collector/processors/sleepprocessor v0.0.1
    path: ./processors/sleepprocessor
  - gomod: github.com/apolzek/sherlock-collector/processors/alertprocessor v0.0.1
    path: ./processors/alertprocessor

exporters:
  - gomod: go.opentelemetry.io/collector/exporter/otlpexporter v0.135.0
  - gomod: go.opentelemetry.io/collector/exporter/debugexporter v0.135.0

```

Build custom-collector
```bash
ocb --config builder-config.yaml
```

Create config.yaml
```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318
        cors:
          allowed_origins:
            - "*"
  
  prometheus:
    config:
      scrape_configs:
        - job_name: 'otel-collector'
          static_configs:
            - targets: ['localhost:8888']

processors:

  sleepprocessor:
    sleep_milliseconds: 10000
  
  alertprocessor:
    webhook_url: "http://localhost:8080/hook" 
    alert_rules:
      - name: "error-detector"
        attribute_key: "error"                   
      
      - name: "critical-severity"
        attribute_key: "severity"
        attribute_value: "critical"               
      
      - name: "database-errors"
        attribute_key: "error.type"
        attribute_value: "database"
        service_name: "user-service"             
      
      - name: "payment-service-monitor"
        service_name: "payment-service"           
      
      - name: "high-priority-alerts"
        attribute_key: "priority"
        attribute_value: "high"                  
        
    webhook_timeout: 10s
    webhook_headers:
      Authorization: "Bearer your-token"
      X-Custom-Header: "otel-alert"
    enabled_for_traces: true
    enabled_for_metrics: true

exporters:

  debug:

service:
  telemetry:
    logs:
      level: "info"

  extensions: []
  
  pipelines:
    traces:
      receivers: [otlp]
      processors: [sleepprocessor,alertprocessor]
      exporters: [debug]
    
    metrics:
      receivers: [prometheus]
      processors: []
      exporters: [debug]
```

Run custom-collector
```bash
./dist/custom-collector --config ./dist/config.yaml
```

Testing
```
curl -X POST http://localhost:4318/v1/traces   -H "Content-Type: application/json"   -d '{
    "resourceSpans": [
      {
        "resource": {
          "attributes": [
            {"key": "service.name", "value": {"stringValue": "api-service"}}
          ]
        },
        "scopeSpans": [
          {
            "scope": {"name": "test-tracer"},
            "spans": [
              {
                "traceId": "22222222222222222222222222222222",
                "spanId": "2222222222222222",
                "name": "critical-operation",
                "kind": 1,
                "startTimeUnixNano": "1695000000000000000",
                "endTimeUnixNano": "1695000001000000000",
                "attributes": [
                  {"key": "severity", "value": {"stringValue": "critical"}},
                  {"key": "operation", "value": {"stringValue": "data-corruption"}}
                ]
              }
            ]
          }
        ]
      }
    ]
  }'
```

### Results

Creating a lightweight version of the collector with ocb is very simple. All you need is a YAML file and the binary, avoiding the default version that may include unnecessary components. Moreover, the project’s architecture makes it easy to build custom components, such as the processors in this PoC. In practice, if you need something not available in the contrib version—whether a receiver, processor, exporter, or connector, you can simply implement your own code. The configuration file then allows you to parameterize and control this behavior. Overall, the ecosystem and its design make it very convenient to extend the collector’s functionality, turning it into a highly strategic component for telemetry.

### References

```
🔗 https://opentelemetry.io/docs/collector/custom-collector/
🔗 https://github.com/open-telemetry/opentelemetry-collector-releases/releases/
🔗 https://github.com/open-telemetry/opentelemetry-collector/tree/main/cmd/builder
🔗 https://opentelemetry.io/docs/collector/installation/
```
