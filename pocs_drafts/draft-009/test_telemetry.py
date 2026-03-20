#!/usr/bin/env python3
"""
Simple script to send test telemetry data to OpenTelemetry Collector
This will generate metrics with service.name attribute to test the count connector
"""

from opentelemetry import metrics
from opentelemetry.sdk.metrics import MeterProvider
from opentelemetry.sdk.metrics.export import PeriodicExportingMetricReader
from opentelemetry.exporter.otlp.proto.grpc.metric_exporter import OTLPMetricExporter
from opentelemetry.sdk.resources import Resource
import time
import random

def send_metrics(service_name, num_metrics=100):
    """Send test metrics to the OTLP collector"""

    # Create a resource with service.name
    resource = Resource(attributes={
        "service.name": service_name,
        "service.version": "1.0.0",
    })

    # Configure OTLP exporter
    otlp_exporter = OTLPMetricExporter(
        endpoint="http://localhost:4317",
        insecure=True
    )

    # Create metric reader and meter provider
    reader = PeriodicExportingMetricReader(otlp_exporter, export_interval_millis=5000)
    provider = MeterProvider(resource=resource, metric_readers=[reader])
    metrics.set_meter_provider(provider)

    # Get a meter
    meter = metrics.get_meter(__name__)

    # Create a counter
    request_counter = meter.create_counter(
        "http.requests",
        description="Number of HTTP requests",
        unit="1"
    )

    # Create a gauge
    temperature_gauge = meter.create_observable_gauge(
        "system.temperature",
        callbacks=[lambda: [(random.uniform(20, 30),)]],
        description="System temperature",
        unit="celsius"
    )

    print(f"Sending {num_metrics} metrics from service: {service_name}")

    # Send metrics
    for i in range(num_metrics):
        request_counter.add(1, {"endpoint": "/api/users", "method": "GET"})
        request_counter.add(1, {"endpoint": "/api/products", "method": "POST"})
        time.sleep(0.1)

    # Wait for final export
    time.sleep(6)
    print(f"Finished sending metrics from {service_name}")

if __name__ == "__main__":
    print("Starting telemetry test...")

    # Send metrics from different services
    services = ["service-alpha", "service-beta", "service-gamma"]

    for service in services:
        send_metrics(service, num_metrics=50)
        print(f"Completed sending data from {service}\n")

    print("Test completed!")
    print("\nYou can now check Prometheus at http://localhost:9090")
    print("Query:   ")                                                                   
