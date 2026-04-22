# otel-tui

Repo: https://github.com/ymtdzzz/otel-tui

A terminal UI that receives OTLP (gRPC/HTTP) traffic locally and lets you
browse traces/metrics/logs interactively — like a zero-dependency Jaeger
replacement for dev/debug.

## What this POC tests

- Pull the image.
- Verify the binary exists and prints help (non-interactive smoke).

## How to run (interactive)

`otel-tui` needs a real TTY — it's a bubbletea TUI. Run it yourself:

```bash
docker run --rm -ti --name otel-tui \
  -p 127.0.0.1:4317:4317 \
  -p 127.0.0.1:4318:4318 \
  ymtdzzz/otel-tui:latest
```

Then point any OTEL SDK/agent at `localhost:4317` (gRPC) or `localhost:4318`
(HTTP). The TUI will render spans as they arrive.

Pair with the `otel-cli` POC next door:

```bash
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317 otel-cli exec --name demo -- sleep 1
```

## What was verified

- `docker pull ymtdzzz/otel-tui:latest` succeeds.
- `docker run ... --help` shows the CLI help, confirming the image entrypoint
  works on this Ubuntu.

## Cleanup

```bash
docker rmi ymtdzzz/otel-tui:latest
```
