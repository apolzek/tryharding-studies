## Processing observability data at scale with Apache Flink

### Objectives

The goal of this PoC is to build a pipeline where a lightweight OpenTelemetry Collector forwards metrics to Kafka without processing logic. Apache Flink consumes those metrics from Kafka and writes the processed data to Prometheus via the Prometheus Sink connector. Processing logic in this PoC focuses on forwarding and normalization — advanced techniques (high cardinality reduction, aggregation, anomaly detection) are left for a follow-up PoC.

### Architecture

```mermaid
graph LR
    A[Applications] --OTLP--> B[OpenTelemetry Collector]
    B --otlp_proto--> C[Kafka]
    C --consume--> D[Apache Flink]
    D --Remote Write--> E[Prometheus]
```

### Services

| Service                 | Port             | Image                                        |
| ----------------------- | ---------------- | -------------------------------------------- |
| zookeeper               | 2181             | confluentinc/cp-zookeeper:7.9.0              |
| kafka                   | 29092            | confluentinc/cp-kafka:7.9.0                  |
| kafka-ui                | 8080             | provectuslabs/kafka-ui:latest                |
| opentelemetry-collector | 4317, 4318, 9115 | otel/opentelemetry-collector-contrib:0.118.0 |
| flink-jobmanager        | 8081, 6123, 9249 | custom (flink-connected.Dockerfile)          |
| flink-taskmanager       | 9250             | custom (flink-connected.Dockerfile)          |
| prometheus              | 9090             | prom/prometheus:v2.45.0                      |
| grafana                 | 3000             | grafana/grafana:latest                       |

### Prerequisites

- docker
- docker compose
- java (openjdk 21+)
- maven

### Reproducing

Build the Flink job JAR
```sh
cd flink-telemetry-processor
mvn clean package
```

Copy the JAR to the jobs directory (mounted into the Flink containers)
```sh
cp flink-telemetry-processor/target/flink-telemetry-processor-1.0-SNAPSHOT.jar flink/jobs/
```

Start the environment
```sh
docker compose up -d
```

Submit the Flink job
```sh
docker exec -it $(docker ps | grep jobmanager | awk '{print $1}') bash
flink run -c com.example.flink.metrics.MetricProcessor jobs/flink-telemetry-processor-1.0-SNAPSHOT.jar 2
```

Send a metric to the OpenTelemetry Collector
```sh
curl -X POST http://localhost:4318/v1/metrics \
  -H "Content-Type: application/json" \
  -d '{
    "resourceMetrics": [{
      "resource": {
        "attributes": [
          {"key": "service.name", "value": {"stringValue": "curl-test-service"}},
          {"key": "service.namespace", "value": {"stringValue": "payment-system"}}
        ]
      },
      "scopeMetrics": [{
        "scope": {"name": "curl.manual.generator", "version": "1.0.0"},
        "metrics": [{
          "name": "test_metric",
          "description": "test metric via curl",
          "unit": "1",
          "sum": {
            "dataPoints": [{
              "asDouble": 42,
              "timeUnixNano": "1691145600000000000",
              "attributes": [
                {"key": "region", "value": {"stringValue": "us-east-1"}},
                {"key": "team", "value": {"stringValue": "observability"}}
              ]
            }],
            "aggregationTemporality": 2,
            "isMonotonic": false
          }
        }]
      }]
    }]
  }'
```

Verify the data flow
- Kafka UI: http://localhost:8080 — check topic `otel-metrics` for incoming messages
- Flink UI: http://localhost:8081 — check the running job and throughput
- Prometheus: http://localhost:9090 — query `test_metric` to confirm remote write arrived
- Grafana: http://localhost:3000 (admin/admin)

### Results

The pipeline works end-to-end: metrics sent via OTLP arrive in Kafka as protobuf-encoded records, Flink deserializes and normalizes them, and writes to Prometheus using the official Prometheus Sink connector. The main complexity lies in deserializing the OTLP protobuf format inside Flink and conforming to the label naming rules required by Prometheus. Offloading processing from the collector to Flink makes the pipeline significantly more scalable and separates concerns between ingestion and transformation.

### References

```
https://grafana.com/blog/2022/10/20/how-to-manage-high-cardinality-metrics-in-prometheus-and-kubernetes/
https://aws.amazon.com/pt/blogs/big-data/process-millions-of-observability-events-with-apache-flink-and-write-directly-to-prometheus/
https://nightlies.apache.org/flink/flink-docs-master/docs/concepts/overview/
https://nightlies.apache.org/flink/flink-docs-master/docs/deployment/overview/#session-mode
https://nightlies.apache.org/flink/flink-docs-master/docs/connectors/datastream/prometheus/
https://github.com/apache/flink-connector-prometheus
https://flink.apache.org/2024/12/05/introducing-the-new-prometheus-connector/
https://prometheus.io/docs/specs/prw/remote_write_spec/
https://github.com/open-telemetry/opentelemetry-proto-java
https://github.com/aws-samples/flink-prometheus-iot-demo
```
