#!/usr/bin/env bash
set -euo pipefail

IMAGE="quay.io/vektorlab/ctop:latest"

docker pull "$IMAGE"
docker run --rm \
  -v /var/run/docker.sock:/var/run/docker.sock:ro \
  "$IMAGE" -v
