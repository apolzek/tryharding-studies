#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"

echo "==> building and starting (primeira build baixa ~2GB de imagem PyTorch + modelo)"
docker compose up -d --build

echo "==> waiting for healthcheck (embedding model precisa carregar e indexar)"
for i in {1..90}; do
  status=$(docker inspect --format='{{.State.Health.Status}}' vec-backend 2>/dev/null || echo "starting")
  if [[ "$status" == "healthy" ]]; then
    echo "   backend healthy"
    break
  fi
  sleep 3
done
if [[ "$status" != "healthy" ]]; then
  echo "!! backend never became healthy"
  docker compose logs backend | tail -40
  exit 1
fi

echo "==> running tests"
docker compose exec -T backend pytest tests/ -v

echo "==> done"
