#!/usr/bin/env bash
# Bring up the observability stack (Prometheus + Grafana + cAdvisor) and the
# external Docker network the benchmark lives on. Idempotent.
set -euo pipefail
ROOT=$(cd "$(dirname "$0")/.." && pwd)
cd "$ROOT"

docker network inspect otelbench >/dev/null 2>&1 || docker network create otelbench >/dev/null
docker compose -f observability/docker-compose.yml up -d

echo ""
echo "Grafana  → http://localhost:3001    (anonymous admin, dashboard: OTel Benchmark)"
echo "Prometheus → http://localhost:9091"
echo "cAdvisor → http://localhost:8081"
echo ""
echo "Next:  ./scripts/sweep.sh grpc 1 512m traces"
echo "   or  ./scripts/run-matrix.sh   # full matrix (~60 min)"
