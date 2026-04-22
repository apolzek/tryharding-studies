# endlessh-go

Repo: https://github.com/shizunge/endlessh-go

SSH tarpit written in Go: accepts connections on port 22 (or whatever you
configure) and trickles out fake SSH banner bytes forever, tying up bots.
Exports Prometheus metrics.

## What this POC tests

- Boot endlessh on `127.0.0.1:19006` with Prometheus metrics on `19007`.
- Make a TCP connection that gets trapped (read one byte, then disconnect).
- Confirm Prometheus exposes `endlessh_client_open_count_total` > 0.

## How to run

```bash
docker compose up -d
./test.sh
docker compose down -v
```

## What was verified

- Tarpit emits a byte every ~2s instead of a real SSH handshake.
- Metrics endpoint serves Prometheus text format.
- `endlessh_client_open_count_total{...}` incremented after our probe.
