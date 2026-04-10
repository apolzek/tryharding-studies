import time
import os
import requests
import json
from datetime import datetime

OTEL_ENDPOINT = os.getenv('OTEL_EXPORTER_OTLP_ENDPOINT', 'http://otel-collector:4318')

def send_json_telemetry():
    """Send telemetry data using HTTP with JSON encoding"""
    url = f"{OTEL_ENDPOINT}/v1/traces"

    # Create a simple trace in JSON format
    trace_data = {
        "resourceSpans": [
            {
                "resource": {
                    "attributes": [
                        {"key": "service.name", "value": {"stringValue": "http-json-sender"}},
                        {"key": "protocol", "value": {"stringValue": "http-json"}}
                    ]
                },
                "scopeSpans": [
                    {
                        "scope": {
                            "name": "http-json-instrumentation",
                            "version": "1.0.0"
                        },
                        "spans": [
                            {
                                "traceId": "5b8efff798038103d269b633813fc60c",
                                "spanId": "eee19b7ec3c1b174",
                                "name": "HTTP-JSON-Operation",
                                "kind": 1,
                                "startTimeUnixNano": str(int(time.time() * 1e9)),
                                "endTimeUnixNano": str(int((time.time() + 0.1) * 1e9)),
                                "attributes": [
                                    {"key": "http.method", "value": {"stringValue": "GET"}},
                                    {"key": "http.url", "value": {"stringValue": "/api/data"}},
                                    {"key": "encoding", "value": {"stringValue": "JSON"}}
                                ]
                            }
                        ]
                    }
                ]
            }
        ]
    }

    headers = {
        'Content-Type': 'application/json'
    }

    try:
        response = requests.post(url, json=trace_data, headers=headers)
        print(f"[{datetime.now()}] Sent JSON telemetry - Status: {response.status_code}")
        print(f"Payload size: {len(json.dumps(trace_data))} bytes")
    except Exception as e:
        print(f"Error sending telemetry: {e}")

def main():
    print("Starting HTTP/JSON telemetry sender...")
    print(f"Sending to: {OTEL_ENDPOINT}")

    time.sleep(5)  # Wait for collector to be ready

    while True:
        send_json_telemetry()
        time.sleep(10)  # Send every 10 seconds

if __name__ == "__main__":
    main()
