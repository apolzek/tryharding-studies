FROM python:3.12-slim

WORKDIR /app

COPY requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt \
 && opentelemetry-bootstrap --action=install

COPY app.py ./

ENV OTEL_SERVICE_NAME=demo-app \
    OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4318 \
    OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf \
    OTEL_METRICS_EXPORTER=otlp \
    OTEL_TRACES_EXPORTER=otlp \
    OTEL_LOGS_EXPORTER=none \
    OTEL_RESOURCE_ATTRIBUTES=service.namespace=tryharding-033,deployment.environment=lab \
    PYTHONUNBUFFERED=1

EXPOSE 8000

CMD ["opentelemetry-instrument", "python", "-m", "flask", "--app", "app", "run", "--host=0.0.0.0", "--port=8000"]
