#!/usr/bin/env bash
set -euo pipefail
CLUSTER=${CLUSTER:-rollouts}
echo "==> deleting kind cluster '${CLUSTER}'"
kind delete cluster --name "$CLUSTER"
