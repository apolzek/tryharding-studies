#!/usr/bin/env bash
set -euo pipefail

BASE="http://127.0.0.1:19002"

echo "==> wait for webhook-tester"
for i in $(seq 1 20); do
  if curl -fsS "$BASE/api/settings" >/dev/null 2>&1; then
    echo "ready"; break
  fi
  sleep 1
done

echo "==> create session"
SID=$(curl -fsS -X POST "$BASE/api/session" \
  -H 'Content-Type: application/json' \
  -d '{"status_code":200,"content_type":"application/json","response_body":"eyJvayI6dHJ1ZX0="}' \
  | python3 -c 'import json,sys;print(json.load(sys.stdin)["uuid"])')
echo "session: $SID"

echo "==> post a payload to capture url"
curl -fsS -X POST "$BASE/$SID" \
  -H 'X-Poc: 047' \
  -H 'Content-Type: application/json' \
  -d '{"hello":"webhook-tester"}' >/dev/null

echo "==> list captured requests"
curl -fsS "$BASE/api/session/$SID/requests" | python3 -m json.tool | head -40
