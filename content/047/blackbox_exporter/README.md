# blackbox_exporter

Repo: https://github.com/prometheus/blackbox_exporter

Prometheus exporter that probes external targets over HTTP, TCP, ICMP, DNS
and exposes the results as metrics. Foundation of most "is-my-endpoint-up"
scrape configs.

## What this POC tests

- Boot blackbox_exporter with a minimal config exposing two modules:
  `http_2xx` and `tcp_connect`.
- Probe two targets from inside the container:
  - `https://example.com` (expect `probe_success=1`)
  - `https://this-host-does-not-exist-047.invalid` (expect `probe_success=0`)

## How to run

```bash
docker compose up -d
./test.sh
docker compose down -v
```

## What was verified

- Happy path: probe_success=1, probe_http_status_code=200 for example.com.
- Failure path: probe_success=0 for the invalid hostname.
