#!/usr/bin/env bash
set -euo pipefail

docker run --rm \
  --network otel-cli-net \
  -e OTEL_EXPORTER_OTLP_ENDPOINT=collector:4317 \
  ghcr.io/equinix-labs/otel-cli:latest \
  exec --endpoint collector:4317 --protocol grpc --insecure \
       --name otel-cli-poc \
       --attrs 'poc=047,tool=otel-cli' \
       -- echo "hello from otel-cli"
