# OpenTelemetry Network Traffic Analysis POC

This POC demonstrates how to analyze network traffic for different OpenTelemetry protocols: HTTP/JSON, HTTP/Protobuf, and gRPC.

## Architecture

The setup includes:
- **OTEL Collector**: Receives telemetry data on ports 4317 (gRPC) and 4318 (HTTP)
- **HTTP/JSON Sender**: Sends telemetry using OTLP HTTP with JSON encoding
- **HTTP/Protobuf Sender**: Sends telemetry using OTLP HTTP with Protobuf encoding
- **gRPC Sender**: Sends telemetry using OTLP gRPC with Protobuf encoding
- **tcpdump Analyzer**: Container with network analysis tools

## Quick Start

### 1. Start the Environment

```bash
docker-compose up -d --build
```

Wait for all containers to start:
```bash
docker-compose ps
```

### 2. View Telemetry Logs

Check that telemetry is being sent and received:

```bash
# View HTTP/JSON sender logs
docker logs -f http-json-sender

# View HTTP/Protobuf sender logs
docker logs -f http-proto-sender

# View gRPC sender logs
docker logs -f grpc-sender

# View OTEL collector logs
docker logs -f otel-collector
```

## Network Traffic Analysis

### Basic tcpdump Commands

Access the tcpdump analyzer container:
```bash
docker exec -it tcpdump-analyzer bash
```

### 1. Capture All OTLP Traffic

```bash
# Capture all traffic on OTLP ports
tcpdump -i any port 4317 or port 4318 -nn -v
```

### 2. Analyze HTTP/JSON Traffic (Port 4318)

```bash
# Capture HTTP traffic with ASCII output
tcpdump -i any port 4318 -A -s 0

# Save to file for later analysis
tcpdump -i any port 4318 -w /tmp/http-json.pcap -s 0

# Read the saved file
tcpdump -r /tmp/http-json.pcap -A
```

### 3. Analyze HTTP/Protobuf Traffic (Port 4318)

```bash
# Capture HTTP Protobuf traffic
tcpdump -i any port 4318 -X -s 0

# Filter by source container (http-proto-sender)
tcpdump -i any port 4318 and src host http-proto-sender -X
```

### 4. Analyze gRPC Traffic (Port 4317)

```bash
# Capture gRPC traffic
tcpdump -i any port 4317 -X -s 0

# Save to pcap file
tcpdump -i any port 4317 -w /tmp/grpc.pcap -s 0
```

### 5. Compare Protocols Side-by-Side

```bash
# Terminal 1: Monitor HTTP traffic
tcpdump -i any port 4318 -nn -v

# Terminal 2: Monitor gRPC traffic
tcpdump -i any port 4317 -nn -v
```

### Advanced tcpdump Options

```bash
# Capture with timestamps
tcpdump -i any port 4318 -tttt -v

# Capture only HTTP POST requests
tcpdump -i any port 4318 -A | grep -A 20 "POST"

# Show packet size statistics
tcpdump -i any port 4317 or port 4318 -nn -q

# Filter by specific HTTP path
tcpdump -i any port 4318 -A | grep "v1/traces"

# Capture with detailed hex dump
tcpdump -i any port 4317 -XX -s 0 -c 10
```

## Protocol Comparison

### HTTP/JSON
- **Port**: 4318
- **Content-Type**: application/json
- **Pros**: Human-readable, easy to debug
- **Cons**: Larger payload size, slower parsing

### HTTP/Protobuf
- **Port**: 4318
- **Content-Type**: application/x-protobuf
- **Pros**: Compact binary format, faster parsing
- **Cons**: Not human-readable, needs protobuf tools to decode

### gRPC
- **Port**: 4317
- **Transport**: HTTP/2
- **Encoding**: Protobuf
- **Pros**: Bidirectional streaming, efficient, multiplexing
- **Cons**: More complex setup, not human-readable

## Analyzing Captured Traffic

### Using tshark (Wireshark CLI)

```bash
# Install tshark in the analyzer container
apt-get update && apt-get install -y tshark

# Analyze HTTP traffic
tshark -i any -f "port 4318" -Y http

# Analyze gRPC traffic with HTTP/2 decoding
tshark -i any -f "port 4317" -Y http2

# Extract HTTP headers
tshark -r /tmp/http-json.pcap -Y http -T fields -e http.request.method -e http.request.uri
```

### Payload Size Comparison

```bash
# Count bytes for HTTP/JSON
tcpdump -i any port 4318 -nn -q -c 10 | awk '{sum+=$NF} END {print "Average:", sum/NR, "bytes"}'

# Count bytes for gRPC
tcpdump -i any port 4317 -nn -q -c 10 | awk '{sum+=$NF} END {print "Average:", sum/NR, "bytes"}'
```

## Sending Manual Telemetry

### Send HTTP/JSON Manually

```bash
curl -X POST http://localhost:4318/v1/traces \
  -H "Content-Type: application/json" \
  -d '{
    "resourceSpans": [{
      "resource": {
        "attributes": [
          {"key": "service.name", "value": {"stringValue": "manual-test"}}
        ]
      },
      "scopeSpans": [{
        "spans": [{
          "traceId": "5b8efff798038103d269b633813fc60c",
          "spanId": "eee19b7ec3c1b174",
          "name": "manual-span",
          "kind": 1,
          "startTimeUnixNano": "1609459200000000000",
          "endTimeUnixNano": "1609459200100000000"
        }]
      }]
    }]
  }'
```

### Send HTTP/Protobuf with Python

```python
from opentelemetry import trace
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter

provider = TracerProvider()
exporter = OTLPSpanExporter(endpoint="http://localhost:4318/v1/traces")
provider.add_span_processor(BatchSpanProcessor(exporter))
trace.set_tracer_provider(provider)

tracer = trace.get_tracer(__name__)
with tracer.start_as_current_span("manual-span"):
    print("Span sent!")
```

## Stopping Individual Senders

To analyze one protocol at a time:

```bash
# Stop HTTP/JSON sender
docker-compose stop http-json-sender

# Stop HTTP/Protobuf sender
docker-compose stop http-proto-sender

# Stop gRPC sender
docker-compose stop grpc-sender

# Restart a sender
docker-compose start http-json-sender
```

## Cleanup

```bash
# Stop all containers
docker-compose down

# Remove all data
docker-compose down -v
```

## Troubleshooting

### Check Container Connectivity

```bash
# From analyzer container
docker exec -it tcpdump-analyzer bash
ping otel-collector
nc -zv otel-collector 4317
nc -zv otel-collector 4318
```

### Check OTEL Collector Status

```bash
docker logs otel-collector | tail -20
```

### Verify Traffic Flow

```bash
# Count packets on each port
docker exec tcpdump-analyzer tcpdump -i any port 4317 -c 10 -nn
docker exec tcpdump-analyzer tcpdump -i any port 4318 -c 10 -nn
```

## Key Observations

When analyzing the traffic, you should notice:

1. **HTTP/JSON**: Plain text payloads, easily readable in tcpdump with `-A` flag
2. **HTTP/Protobuf**: Binary payloads, more compact, requires protobuf decoding
3. **gRPC**: Binary HTTP/2 frames, most efficient, requires HTTP/2 analysis tools

## Additional Resources

- [OTLP Specification](https://opentelemetry.io/docs/specs/otlp/)
- [OpenTelemetry Protocol](https://github.com/open-telemetry/opentelemetry-proto)
- [tcpdump Tutorial](https://www.tcpdump.org/manpages/tcpdump.1.html)
