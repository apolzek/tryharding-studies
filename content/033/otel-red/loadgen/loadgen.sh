#!/bin/sh
# Weighted traffic generator. Hit patterns mean the MCP-driven answers
# below are easy to sanity-check:
#   - /api/fast    ~ 5-20ms    (dominant volume)
#   - /api/medium  ~ 50-150ms
#   - /api/slow    ~ 400-1200ms -> will own p95
#   - /api/flaky   ~35% 5xx     -> drives error rate
#   - /api/notfound always 404
set -eu
BASE="${TARGET:-http://app:8000}"

i=0
while :; do
  # 5x fast, 2x medium, 1x slow per outer iteration, plus flaky/notfound bursts
  for _ in 1 2 3 4 5; do wget -q -O- "$BASE/api/fast"    >/dev/null 2>&1 || true; done
  for _ in 1 2;         do wget -q -O- "$BASE/api/medium"  >/dev/null 2>&1 || true; done
  wget -q -O- "$BASE/api/slow" >/dev/null 2>&1 || true
  wget -q -O- "$BASE/api/flaky" >/dev/null 2>&1 || true
  if [ $((i % 7)) -eq 0 ]; then
    wget -q -O- "$BASE/api/notfound" >/dev/null 2>&1 || true
  fi
  i=$((i + 1))
  sleep 0.5
done
