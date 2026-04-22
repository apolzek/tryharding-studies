#!/usr/bin/env bash
set -euo pipefail

BASE="http://127.0.0.1:19005"

echo "==> wait for blackbox_exporter"
for i in $(seq 1 15); do
  if curl -fsS "$BASE/-/healthy" >/dev/null 2>&1; then echo "ready"; break; fi
  sleep 1
done

echo "==> probe example.com (http_2xx)"
curl -fsS "$BASE/probe?module=http_2xx&target=https://example.com" \
  | grep -E '^probe_(success|http_status_code|duration_seconds) ' \
  | head -10

echo
echo "==> probe invalid host (http_2xx) — should fail"
curl -fsS "$BASE/probe?module=http_2xx&target=https://this-host-does-not-exist-047.invalid" \
  | grep -E '^probe_(success|dns_lookup_time_seconds) ' \
  | head -10
