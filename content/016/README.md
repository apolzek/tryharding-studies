## Routing OTLP Data to Kafka with Vector

### Objectives

The goal of this PoC is to evaluate Vector as an alternative to the OpenTelemetry Collector for receiving OTLP telemetry and routing it to Kafka. Vector's `opentelemetry` source accepts both gRPC and HTTP OTLP traffic, and the `kafka` sink exports traces as JSON to a Kafka topic. This PoC is exploratory — the Vector service in `docker-compose.yaml` is commented out and the pipeline is not fully validated end-to-end.

### Architecture

```mermaid
graph LR
    APP[Application] --OTLP gRPC:4319 / HTTP:4320--> VT[Vector]
    VT --traces_to_logs remap--> KF[Kafka topic: topic-example]
    VT --file sink--> FS[/tmp/vector]
```

### Services

| Service | Port             | Image                         |
| ------- | ---------------- | ----------------------------- |
| vector  | 4319, 4320, 8686 | timberio/vector:0.50.0-debian |
| kafka   | 29092            | confluentinc/cp-kafka         |
| zookeeper | 2181           | confluentinc/cp-zookeeper     |

Note: the Vector service is commented out in `docker-compose.yaml`. Uncomment it to run.

### Prerequisites

- docker
- docker compose

### Reproducing

Uncomment the `vector` service in `docker-compose.yaml`, then start the stack

```sh
docker compose up -d
```

Send a test trace to the Vector OTLP HTTP endpoint

```sh
curl -X POST http://localhost:4320/v1/traces \
  -H "Content-Type: application/json" \
  -d '{
    "resourceSpans": [{
      "resource": {
        "attributes": [{"key": "service.name", "value": {"stringValue": "test-service"}}]
      },
      "scopeSpans": [{
        "spans": [{
          "traceId": "5b8efff798038103d269b633813fc60c",
          "spanId": "eee19b7ec3c1b174",
          "name": "test-span",
          "kind": 1,
          "startTimeUnixNano": "1640995200000000000",
          "endTimeUnixNano": "1640995201000000000"
        }]
      }]
    }]
  }'
```

Check Vector API for pipeline health

```sh
curl http://localhost:8686/health
```

Verify Kafka received the message by checking the `topic-example` topic.

### Results

Vector's `opentelemetry` source supports OTLP over both gRPC and HTTP natively. The `remap` transform converts the trace structure to a flat JSON string before the Kafka sink serializes it. The pipeline is a viable alternative to OTel Collector for simple OTLP-to-Kafka routing, with less configuration overhead. This PoC did not reach a fully running state — the Vector service needs to be uncommented and tested against a live Kafka instance to confirm end-to-end delivery.

### References

```
https://vector.dev/docs/reference/configuration/sources/opentelemetry/
https://vector.dev/docs/reference/configuration/sinks/kafka/
https://vector.dev/docs/reference/vrl/
```
