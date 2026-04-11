FROM python:3.12-slim

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY main.py .

EXPOSE 8082

# opentelemetry-instrument reads OTEL_* env vars and auto-instruments
# FastAPI, httpx and asyncpg before the process starts.
CMD ["opentelemetry-instrument", "uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8082"]
