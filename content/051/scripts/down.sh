#!/usr/bin/env bash
# Tear everything down so no container is left running.
set -euo pipefail
ROOT=$(cd "$(dirname "$0")/.." && pwd)
cd "$ROOT"

docker compose -f otlp-grpc/docker-compose.yml down --remove-orphans 2>/dev/null || true
docker compose -f otlp-http/docker-compose.yml down --remove-orphans 2>/dev/null || true
docker compose -f observability/docker-compose.yml down --remove-orphans 2>/dev/null || true
docker ps -a --filter "name=tg-" -q | xargs -r docker rm -f 2>/dev/null || true
docker network rm otelbench 2>/dev/null || true
echo "done"
