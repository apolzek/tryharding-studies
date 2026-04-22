#!/usr/bin/env bash
set -euo pipefail

IMAGE="ymtdzzz/otel-tui:latest"

docker pull "$IMAGE"
docker run --rm --entrypoint /otel-tui "$IMAGE" --help
