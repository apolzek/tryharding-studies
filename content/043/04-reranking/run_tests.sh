#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
echo "==> building and starting (1ª vez baixa bge-reranker-v2-m3 ~2.2GB)"
docker compose up -d --build
for i in {1..120}; do
  status=$(docker inspect --format='{{.State.Health.Status}}' rrk-backend 2>/dev/null || echo "starting")
  [[ "$status" == "healthy" ]] && break
  sleep 5
done
if [[ "$status" != "healthy" ]]; then
  echo "!! backend never became healthy"
  docker compose logs backend | tail -40
  exit 1
fi
echo "==> running tests"
docker compose exec -T backend pytest tests/ -v
echo "==> done"
