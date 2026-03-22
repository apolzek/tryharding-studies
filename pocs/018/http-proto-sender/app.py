import time
import os
from opentelemetry import trace
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import Resource
from datetime import datetime

def setup_telemetry():
    """Setup OpenTelemetry with HTTP/Protobuf exporter"""
    resource = Resource.create({
        "service.name": "http-proto-sender",
        "protocol": "http-protobuf"
    })

    provider = TracerProvider(resource=resource)

    # OTLP HTTP exporter with protobuf encoding (default)
    otlp_exporter = OTLPSpanExporter(
        endpoint=f"{os.getenv('OTEL_EXPORTER_OTLP_ENDPOINT', 'http://otel-collector:4318')}/v1/traces",
    )

    processor = BatchSpanProcessor(otlp_exporter)
    provider.add_span_processor(processor)

    trace.set_tracer_provider(provider)

    return trace.get_tracer(__name__)

def main():
    print("Starting HTTP/Protobuf telemetry sender...")
    print(f"Sending to: {os.getenv('OTEL_EXPORTER_OTLP_ENDPOINT', 'http://otel-collector:4318')}")

    time.sleep(5)  # Wait for collector to be ready

    tracer = setup_telemetry()

    while True:
        with tracer.start_as_current_span("HTTP-Protobuf-Operation") as span:
            span.set_attribute("http.method", "POST")
            span.set_attribute("http.url", "/api/process")
            span.set_attribute("encoding", "Protobuf")
            span.set_attribute("protocol", "HTTP")

            # Simulate some work
            time.sleep(0.1)

            print(f"[{datetime.now()}] Sent Protobuf telemetry via HTTP")

        time.sleep(10)  # Send every 10 seconds

if __name__ == "__main__":
    main()
