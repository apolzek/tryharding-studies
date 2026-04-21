#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
echo "==> building and starting (1ª vez: baixa Ollama + qwen2.5:3b ~2GB + reranker ~2.2GB)"
docker compose up -d --build
echo "==> aguardando backend ficar healthy (pode levar 5-10min na 1ª execução)"
for i in {1..240}; do
  status=$(docker inspect --format='{{.State.Health.Status}}' rag-backend 2>/dev/null || echo "starting")
  [[ "$status" == "healthy" ]] && break
  sleep 5
done
if [[ "$status" != "healthy" ]]; then
  echo "!! backend never became healthy"
  docker compose logs backend | tail -40
  exit 1
fi
echo "==> running tests (LLM runs are ~5-15s each; dá um café)"
docker compose exec -T backend pytest tests/ -v --timeout=300
echo "==> done"
