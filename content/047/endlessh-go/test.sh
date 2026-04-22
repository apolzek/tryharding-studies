#!/usr/bin/env bash
set -euo pipefail

METRICS="http://127.0.0.1:19007/metrics"

echo "==> wait for metrics"
for i in $(seq 1 15); do
  if curl -fsS "$METRICS" >/dev/null 2>&1; then echo "ready"; break; fi
  sleep 1
done

echo "==> connect to tarpit and read up to 16 bytes"
# We give it 6 seconds so we get >= 2 trickle chunks at 2s interval
timeout 6 bash -c 'exec 3<>/dev/tcp/127.0.0.1/19006; head -c 16 <&3' | xxd | head -2 || true

echo
echo "==> scrape relevant metrics"
curl -fsS "$METRICS" | grep -E '^endlessh_client_(open_count|trapped_time)' | head -10
