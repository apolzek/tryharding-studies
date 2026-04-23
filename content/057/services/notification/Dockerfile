FROM python:3.12-slim
WORKDIR /app
ENV PYTHONDONTWRITEBYTECODE=1 PYTHONUNBUFFERED=1

COPY requirements.txt .
# setuptools>=81 removed pkg_resources, which opentelemetry-bootstrap still imports.
RUN pip install --no-cache-dir --upgrade pip "setuptools<80" wheel && \
    pip install --no-cache-dir -r requirements.txt && \
    opentelemetry-bootstrap -a install

COPY app ./app
COPY tests ./tests

EXPOSE 8008
CMD ["opentelemetry-instrument", \
     "--service_name", "notification", \
     "--traces_exporter", "otlp", \
     "--metrics_exporter", "otlp", \
     "uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8008"]
