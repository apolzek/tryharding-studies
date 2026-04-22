#!/usr/bin/env bash
# Orchestrate full load-test matrix: 5 resource tiers × 2 protocols.
# Results are appended to results/*.csv by each sweep.
set -euo pipefail

ROOT=$(cd "$(dirname "$0")/.." && pwd)
cd "$ROOT"

# Reset results
rm -f results/*.csv
echo "protocol,cpus,mem,signal,best_sustained_rate" > results/best.csv

TIERS=(
  "0.5 256m"
  "1   512m"
  "1   1g"
  "2   2g"
  "2   4g"
)

# Ensure the observability stack is up + network ready
docker network inspect otelbench >/dev/null 2>&1 || docker network create otelbench >/dev/null
docker compose -f observability/docker-compose.yml up -d >/dev/null

START=$(date +%s)

for tier in "${TIERS[@]}"; do
  read -r cpus mem <<<"$tier"
  for proto in grpc http; do
    echo ""
    echo "##########################################################"
    echo "#  tier=${cpus}c/${mem}  proto=${proto}   elapsed=$(( ($(date +%s) - START) / 60 ))m"
    echo "##########################################################"
    bash scripts/sweep.sh "$proto" "$cpus" "$mem" traces
  done
done

echo ""
echo "=== MATRIX SUMMARY ==="
cat results/best.csv | column -t -s ,
echo ""
echo "Total duration: $(( ($(date +%s) - START) / 60 ))m"
