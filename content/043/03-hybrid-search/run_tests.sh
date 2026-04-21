#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
echo "==> building and starting"
docker compose up -d --build
for i in {1..90}; do
  status=$(docker inspect --format='{{.State.Health.Status}}' hyb-backend 2>/dev/null || echo "starting")
  [[ "$status" == "healthy" ]] && break
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
