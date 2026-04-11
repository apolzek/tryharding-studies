#!/bin/sh
set -e

TOXIPROXY_URL="http://toxiproxy:8474"
LATENCY_MS=100
JITTER_MS=20

echo "==> Waiting for Toxiproxy API to be available..."
until curl -sf "${TOXIPROXY_URL}/proxies" > /dev/null 2>&1; do
  echo "    Toxiproxy not ready yet, retrying in 2s..."
  sleep 2
done

echo "==> Toxiproxy is up. Configuring latency toxic..."

RESPONSE=$(curl -sf -X POST "${TOXIPROXY_URL}/proxies/otel-collector/toxics" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"latency\",
    \"type\": \"latency\",
    \"stream\": \"upstream\",
    \"toxicity\": 1.0,
    \"attributes\": {
      \"latency\": ${LATENCY_MS},
      \"jitter\": ${JITTER_MS}
    }
  }")

echo "==> Toxic created: ${RESPONSE}"
echo "==> Done! Latency of ${LATENCY_MS}ms ± ${JITTER_MS}ms applied on otel-collector proxy."
