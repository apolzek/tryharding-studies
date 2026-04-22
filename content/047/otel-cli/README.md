# otel-cli

Repo: https://github.com/equinix-labs/otel-cli

CLI tool to emit OpenTelemetry traces from shell scripts. Useful for
instrumenting legacy cron jobs, CI pipelines, etc.

## What this POC tests

1. Start a local OTEL collector (`otel/opentelemetry-collector-contrib`) that
   logs everything it receives.
2. Use the `otel-cli` image to emit a span via OTLP/gRPC to the collector.
3. Confirm the span landed in the collector logs.

## How to run

```bash
docker compose up -d collector
./send-span.sh
docker logs otel-cli-collector --tail 50
docker compose down -v
```

## What was verified

- `otel-cli exec -- echo hi` sends a span over OTLP gRPC to the collector.
- The collector's `debug` exporter prints the span name
  (`otel-cli-poc`) and the attributes set via `--attrs`.

## Notes

- Collector ports kept off the common 4317/4318 to avoid clashing with other
  OTEL stacks on this machine. Default is kept because only short-lived
  containers share a private bridge network.
