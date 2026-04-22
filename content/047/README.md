# 047 — Docker POC batch from bookmarks

> One-line summary: 11 GitHub tools from my bookmarks, each running in Docker,
> each smoke-tested end-to-end, each documented and torn down cleanly.

This folder is a self-contained batch of Docker proof-of-concepts I pulled
from `~/Workdir/apolzek/neosearch/bookmarks/` (yaml + txt files indexing
~280 unique GitHub repos). Every POC lives in its own folder with three
things: a `docker-compose.yml` (or `smoke.sh`), a `README.md`, and — when the
tool has a usable HTTP/gRPC API — a `test.sh` that exercises it.

---

## Table of contents

1. [Goals and non-goals](#goals-and-non-goals)
2. [Prerequisites](#prerequisites)
3. [Quickstart](#quickstart)
4. [Inventory of POCs](#inventory-of-pocs)
5. [Port map](#port-map)
6. [Directory layout](#directory-layout)
7. [Conventions](#conventions)
8. [Troubleshooting](#troubleshooting)
9. [Cleanup](#cleanup)
10. [What was not tested](#what-was-not-tested)
11. [Further reading](#further-reading)

---

## Goals and non-goals

**Goals**

- Exercise a diverse slice of the bookmarked tools on *my* Ubuntu box,
  fast, without polluting the host.
- Produce a repeatable POC: someone can `cd <tool>/ && docker compose up -d`
  and replay the exact test I ran.
- Give honest evidence: each POC's README shows *what* was verified, not just
  "it started".

**Non-goals**

- Build a long-running platform. Everything here is ephemeral.
- Cover every bookmarked repo. See [`skip.md`](./skip.md) for what was
  rejected, organized by reason.
- Evaluate production readiness. These are *smoke tests*, not benchmarks.

## Prerequisites

| Requirement | Why |
|-------------|-----|
| Docker Engine ≥ 24 | All tools ship via Docker |
| Docker Compose v2 (`docker compose`) | Most POCs use a compose file |
| `curl`, `python3`, `xxd` | `test.sh` scripts use them |
| ~2 GB free disk | For pulled images (peak; volumes are cleaned per-POC) |
| Loopback ports `19001-19008` free | See [Port map](#port-map) |

Verify your box:

```bash
docker --version
docker compose version
ss -tln | awk '{print $4}' | grep -E ':(19001|19002|19003|19004|19005|19006|19007|19008)$' \
  && echo "collision — free these ports" || echo "ports free"
```

## Quickstart

Pick any folder and run the three-liner. Example with `uptime-kuma`:

```bash
cd uptime-kuma/
docker compose up -d
curl -sI http://127.0.0.1:19001/ | head -1     # expect 302 Found
docker compose down -v
```

A few POCs aren't compose-based (pure CLI tools that emit one-shot output);
those ship a `smoke.sh` instead:

```bash
cd ctop/
./smoke.sh            # pulls image, prints version
```

And some POCs have both a compose + a `test.sh` that drives the API:

```bash
cd webhook-tester/
docker compose up -d
./test.sh
docker compose down -v
```

## Inventory of POCs

| # | Tool | Category | Folder | Runner | What the POC verifies |
|---|------|----------|--------|--------|------------------------|
| 1 | [`bcicen/ctop`](https://github.com/bcicen/ctop) | container ops | [`ctop/`](./ctop/) | `smoke.sh` | pulls image, `ctop -v` prints version through the mounted docker socket |
| 2 | [`equinix-labs/otel-cli`](https://github.com/equinix-labs/otel-cli) | observability | [`otel-cli/`](./otel-cli/) | compose + `send-span.sh` | emits a span over OTLP/gRPC, collector `debug` exporter prints it |
| 3 | [`ymtdzzz/otel-tui`](https://github.com/ymtdzzz/otel-tui) | observability | [`otel-tui/`](./otel-tui/) | `smoke.sh` | CLI `--help` works (the TUI itself needs a real TTY; docs show how) |
| 4 | [`louislam/uptime-kuma`](https://github.com/louislam/uptime-kuma) | monitoring | [`uptime-kuma/`](./uptime-kuma/) | compose | web UI boots, `GET /dashboard` → 200 + `<title>Uptime Kuma</title>` |
| 5 | [`tarampampam/webhook-tester`](https://github.com/tarampampam/webhook-tester) | dev-tool | [`webhook-tester/`](./webhook-tester/) | compose + `test.sh` | creates a capture session, POST is captured with body + `X-Poc` header |
| 6 | [`dutchcoders/transfer.sh`](https://github.com/dutchcoders/transfer.sh) | dev-tool | [`transfer.sh/`](./transfer.sh/) | compose + `test.sh` | upload a 4KiB random blob and re-download it — sha256 matches |
| 7 | [`zincsearch/zincsearch`](https://github.com/zincsearch/zincsearch) | search | [`zincsearch/`](./zincsearch/) | compose + `test.sh` | bulk-insert 3 docs into `poc047`, search returns the expected hit |
| 8 | [`prometheus/blackbox_exporter`](https://github.com/prometheus/blackbox_exporter) | observability | [`blackbox_exporter/`](./blackbox_exporter/) | compose + `test.sh` | `probe_success=1` for example.com; `=0` for a non-existent host |
| 9 | [`shizunge/endlessh-go`](https://github.com/shizunge/endlessh-go) | security | [`endlessh-go/`](./endlessh-go/) | compose + `test.sh` | tarpit trickles bytes at 2s intervals; metric `trapped_time_seconds=4.0+` |
| 10 | [`clemcer/LoggiFly`](https://github.com/clemcer/LoggiFly) | observability | [`loggifly/`](./loggifly/) | compose + `test.sh` | attaches to the docker socket and flags a `critical` keyword in a trigger container |
| 11 | [`AykutSarac/jsoncrack.com`](https://github.com/AykutSarac/jsoncrack.com) | dev-tool | [`jsoncrack/`](./jsoncrack/) | compose | static app responds 200, page title `"JSON Crack | Online JSON Viewer..."` |

## Port map

All ports are bound to `127.0.0.1` (loopback) and land in the `19000` block
to avoid collision with the `rag-*`, `rrk-*`, `hyb-*`, `vec-*`, `kw-*` stacks
already running on this machine.

| Host port | Service | Protocol | Container port |
|-----------|---------|----------|----------------|
| 19001 | uptime-kuma | HTTP | 3001 |
| 19002 | webhook-tester | HTTP | 8080 |
| 19003 | transfer.sh | HTTP | 8080 |
| 19004 | zincsearch | HTTP | 4080 |
| 19005 | blackbox_exporter | HTTP | 9115 |
| 19006 | endlessh (tarpit) | TCP | 2222 |
| 19007 | endlessh (metrics) | HTTP | 2112 |
| 19008 | jsoncrack | HTTP | 8080 |

`ctop`, `otel-cli`, `otel-tui`, `loggifly` are headless or connect to a
private network — no host port published.

## Directory layout

```
content/047/
├── README.md                 # this file
├── skip.md                   # what was not tested, by reason
├── FINDINGS.md               # gotchas discovered while running the POCs
├── CONTRIBUTING.md           # how to add a new POC to this batch
│
├── ctop/
│   ├── README.md
│   └── smoke.sh
│
├── otel-cli/
│   ├── README.md
│   ├── docker-compose.yml
│   ├── otel-collector.yaml
│   └── send-span.sh
│
├── otel-tui/
│   ├── README.md
│   └── smoke.sh
│
├── uptime-kuma/
│   ├── README.md
│   └── docker-compose.yml
│
├── webhook-tester/
│   ├── README.md
│   ├── docker-compose.yml
│   └── test.sh
│
├── transfer.sh/
│   ├── README.md
│   ├── docker-compose.yml
│   └── test.sh
│
├── zincsearch/
│   ├── README.md
│   ├── docker-compose.yml
│   └── test.sh
│
├── blackbox_exporter/
│   ├── README.md
│   ├── docker-compose.yml
│   ├── blackbox.yml
│   └── test.sh
│
├── endlessh-go/
│   ├── README.md
│   ├── docker-compose.yml
│   └── test.sh
│
├── loggifly/
│   ├── README.md
│   ├── docker-compose.yml
│   └── test.sh
│
└── jsoncrack/
    ├── README.md
    └── docker-compose.yml
```

## Conventions

Documented in detail in [`CONTRIBUTING.md`](./CONTRIBUTING.md), summarized:

- **One folder per tool.** Name matches the repo name (lowercased, dashes).
- **Ports on loopback** (`127.0.0.1:...`) in the `19xxx` range.
- **Volumes declared in-compose**, cleaned with `down -v` after the test.
- **Read-only mounts** for docker sockets and config files.
- **README sections** follow a stable template: *What this POC tests →
  How to run → What was verified → Notes.*

## Troubleshooting

See also [`FINDINGS.md`](./FINDINGS.md) for per-tool pitfalls I hit.

| Symptom | Likely cause | Fix |
|---------|--------------|-----|
| `Error pull access denied` | Image moved or never existed on Docker Hub | Check the repo's README for the current registry path (ghcr.io, quay.io, public ECR) |
| Container starts then restarts | Flag/env unknown to the binary | `docker logs <name>` — look for `flag provided but not defined` |
| `curl` hangs or returns 000 | Wrong internal port, or image didn't bind yet | `docker port <name>`; `docker exec <name> sh -c 'ss -tln'` |
| Port collision | Another stack already bound `19xxx` | `ss -tln \| grep 19`; remap in the compose file |
| Compose up said "pulling" forever | Network or rate-limit issue | Try `docker pull` manually first; on Docker Hub anon-limit, authenticate |
| `/var/run/docker.sock: permission denied` | User not in `docker` group | Either `newgrp docker` or run compose with `sudo` (avoid in prod) |

## Cleanup

Safe batch-cleanup after you're done playing:

```bash
# from content/047/
for d in */; do
  [ -f "$d/docker-compose.yml" ] && ( cd "$d" && docker compose down -v )
done
# optional: drop images we pulled
docker rmi \
  quay.io/vektorlab/ctop:latest \
  otel/opentelemetry-collector-contrib:0.110.0 \
  ghcr.io/equinix-labs/otel-cli:latest \
  ymtdzzz/otel-tui:latest \
  louislam/uptime-kuma:1 \
  ghcr.io/tarampampam/webhook-tester:2 \
  dutchcoders/transfer.sh:latest \
  public.ecr.aws/zinclabs/zincsearch:latest \
  prom/blackbox-exporter:v0.25.0 \
  shizunge/endlessh-go:latest \
  ghcr.io/clemcer/loggifly:v1 \
  gorkinus/jsoncrack:latest 2>/dev/null || true
```

## What was not tested

See [`skip.md`](./skip.md). ~270 of the ~280 bookmarked repos were skipped,
grouped by reason — awesome-lists, heavy ML, giant platforms, TTY-only TUIs,
hardware-specific, dual-use offensive tooling, broken URLs, etc. That file
also has a round-2 shortlist (hetty, opengrep, katana, toxiproxy, …) worth
doing next.

## Further reading

- [`skip.md`](./skip.md) — rejected repos, by category.
- [`FINDINGS.md`](./FINDINGS.md) — concrete gotchas discovered while running
  these POCs (flag typos, port drift, entrypoint quirks).
- [`CONTRIBUTING.md`](./CONTRIBUTING.md) — drop-in template for adding a new
  POC that matches the conventions above.
