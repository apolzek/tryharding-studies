#!/usr/bin/env bash
# Tears the POC down. Keeps generated certs unless --all is passed.
set -euo pipefail
kind delete cluster --name east || true
kind delete cluster --name west || true
if [[ "${1:-}" == "--all" ]]; then
  rm -rf "$(dirname "$0")/certs" "$(dirname "$0")/bin"
fi
