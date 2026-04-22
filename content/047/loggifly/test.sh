#!/usr/bin/env bash
set -euo pipefail

echo "==> wait 6s for loggifly to attach"
sleep 6

echo "==> loggifly boot logs"
docker logs loggifly-poc 2>&1 | tail -25
echo

echo "==> trigger a matching log line from a throwaway container"
docker run --rm --name loggifly-poc-trigger alpine \
  sh -c 'echo "critical: poc-047 match-me"'

echo
echo "==> wait 4s for loggifly to process"
sleep 4

echo "==> loggifly logs after trigger (grep for match / keyword)"
docker logs loggifly-poc 2>&1 | grep -iE 'match|keyword|critical|poc-047' | tail -10 || true
