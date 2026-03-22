import time
import os
from opentelemetry import trace
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import Resource
from datetime import datetime

def setup_telemetry():
    """Setup OpenTelemetry with gRPC exporter"""
    resource = Resource.create({
        "service.name": "grpc-sender",
        "protocol": "grpc"
    })

    provider = TracerProvider(resource=resource)

    # OTLP gRPC exporter
    otlp_exporter = OTLPSpanExporter(
        endpoint=os.getenv('OTEL_EXPORTER_OTLP_ENDPOINT', 'http://otel-collector:4317').replace('http://', ''),
        insecure=True
    )

    processor = BatchSpanProcessor(otlp_exporter)
    provider.add_span_processor(processor)

    trace.set_tracer_provider(provider)

    return trace.get_tracer(__name__)

def main():
    print("Starting gRPC telemetry sender...")
    print(f"Sending to: {os.getenv('OTEL_EXPORTER_OTLP_ENDPOINT', 'http://otel-collector:4317')}")

    time.sleep(5)  # Wait for collector to be ready

    tracer = setup_telemetry()

    while True:
        with tracer.start_as_current_span("gRPC-Operation") as span:
            span.set_attribute("http.method", "GET")
            span.set_attribute("http.url", "/api/stream")
            span.set_attribute("encoding", "Protobuf")
            span.set_attribute("protocol", "gRPC")

            # Simulate some work
            time.sleep(0.1)

            print(f"[{datetime.now()}] Sent Protobuf telemetry via gRPC")

        time.sleep(10)  # Send every 10 seconds

if __name__ == "__main__":
    main()
