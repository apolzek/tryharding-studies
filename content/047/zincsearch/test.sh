#!/usr/bin/env bash
set -euo pipefail

BASE="http://127.0.0.1:19004"
AUTH="admin:Admin123!"

echo "==> wait for zincsearch"
for i in $(seq 1 30); do
  if curl -fsS -u "$AUTH" "$BASE/version" >/dev/null 2>&1; then
    echo "ready"; break
  fi
  sleep 1
done

echo "==> version"
curl -fsS -u "$AUTH" "$BASE/version"
echo

echo "==> bulk insert into index 'poc047'"
curl -fsS -u "$AUTH" -X POST "$BASE/api/_bulkv2" \
  -H 'Content-Type: application/json' \
  -d '{
    "index":"poc047",
    "records":[
      {"title":"rinha de backend 2025","tag":"benchmark"},
      {"title":"otel-cli poc","tag":"observability"},
      {"title":"uptime kuma","tag":"monitoring"}
    ]
  }'
echo

sleep 1

echo "==> search 'rinha'"
curl -fsS -u "$AUTH" -X POST "$BASE/api/poc047/_search" \
  -H 'Content-Type: application/json' \
  -d '{"search_type":"match","query":{"term":"rinha","field":"title"}}' \
  | python3 -m json.tool
