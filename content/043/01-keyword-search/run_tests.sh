#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"

echo "==> building and starting"
docker compose up -d --build

echo "==> waiting for healthcheck"
for i in {1..30}; do
  status=$(docker inspect --format='{{.State.Health.Status}}' kw-backend 2>/dev/null || echo "starting")
  if [[ "$status" == "healthy" ]]; then
    echo "   backend is healthy"
    break
  fi
  sleep 2
done
if [[ "$status" != "healthy" ]]; then
  echo "!! backend never became healthy"
  docker compose logs backend
  exit 1
fi

echo "==> running tests"
docker compose exec -T backend pytest tests/ -v

echo "==> done — leaving stack up (run 'docker compose down' when finished)"
