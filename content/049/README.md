# 049 — BigBench: one Go API, ten load-test tools

A tiny Go HTTP service with three endpoints of increasing algorithmic
complexity, benchmarked end-to-end with **10 different load-testing tools**,
all dockerized with `--network host` to avoid port conflicts and maximize
TCP throughput (no docker-proxy / NAT hop).

---

## 1. The application

`app/main.go` exposes four endpoints on **`:8765`**:

| Endpoint  | Complexity     | What it does                                              |
| --------- | -------------- | --------------------------------------------------------- |
| `/health` | O(1)           | Liveness probe                                            |
| `/simple` | **O(1)**       | Ten arithmetic ops, constant time                         |
| `/medium` | **O(n log n)** | Generate n random floats and sort them (default n=2000)   |
| `/heavy`  | **O(2^n)**     | Naïve recursive Fibonacci (default n=25, capped at n=35)  |

All responses are JSON; every response includes `elapsed_ms` so you can see
the server-side cost of each complexity class. The container is built from a
multi-stage `golang:1.22-alpine` → `distroless/static-debian12` image, runs
as a non-root user, and listens directly on the host via `network_mode: host`.

### Why host networking?

- **Removes NAT overhead.** Docker's userland `docker-proxy` and bridge NAT
  cap small-request throughput well below what the kernel can do. Host
  networking gives each load tool the same raw path as if the server were a
  local process.
- **No port conflicts.** Every load tool connects to `127.0.0.1:8765`
  directly — there are no published ports to collide, and tools can run
  sequentially or in parallel without bridge-mode port contention.
- **Zero local installs.** Every load tool also runs under `--network host`,
  so the host only needs `docker` + `go` + `make`.

---

## 2. Layout

```
049/
├── app/                 # Go service + Dockerfile
│   ├── main.go
│   ├── go.mod
│   └── Dockerfile
├── loadtests/           # per-tool configs
│   ├── k6/script.js
│   ├── locust/locustfile.py
│   ├── vegeta/targets.txt
│   ├── artillery/config.yml
│   ├── drill/benchmark.yml
│   ├── siege/urls.txt
│   └── wrk/wrk.lua
├── results/             # raw tool output (created by `make test-*`)
├── docker-compose.yml   # host-network service
├── Makefile             # all orchestration
└── README.md
```

---

## 3. Quick start

```bash
make build         # build bigbench:local image
make up            # start container, wait until /health is green
make smoke         # curl the three endpoints and show JSON
make test-all      # run all 10 tools (~4 minutes total)
make down          # tear it down
```

Individual tools:

```bash
make test-k6
make test-vegeta
make test-wrk
make test-hey
make test-ab
make test-bombardier
make test-siege
make test-artillery
make test-locust
make test-drill
```

Tuning knobs (override on the command line):

```bash
make test-k6 DUR=30s VUS=100 RPS=2000
```

---

## 4. The ten tools

Every tool runs as an **ephemeral container with `--network host`**. No tool
is installed on the host — tearing down the sandbox leaves nothing behind.

| # | Tool        | Language | Model                    | Docker image                 |
| - | ----------- | -------- | ------------------------ | ---------------------------- |
| 1 | k6          | Go / JS  | VU-based scripting       | `grafana/k6:0.50.0`          |
| 2 | Vegeta      | Go       | Constant-rate attack     | `peterevans/vegeta:latest`   |
| 3 | wrk         | C + Lua  | Multi-threaded epoll     | `williamyeh/wrk:latest`      |
| 4 | hey         | Go       | Simple closed-loop       | `rcmorano/docker-hey`        |
| 5 | ApacheBench | C        | Single-thread concurrent | `httpd:2.4-alpine` (`ab`)    |
| 6 | Bombardier  | Go       | Fasthttp-based           | `alpine/bombardier:latest`   |
| 7 | Siege       | C        | HTTP "regression" tester | `jstarcher/siege:latest`     |
| 8 | Artillery   | Node.js  | Scenario/arrival-rate    | `artilleryio/artillery:latest` |
| 9 | Locust      | Python   | Distributed users        | `locustio/locust:2.24.0`     |
| 10| Drill       | Rust     | YAML-described flows     | `xridge/drill:latest`        |

---

## 5. Results

Test environment on `2026-04-22`:

- Host: Linux 6.17, Docker 29.4.1, Go 1.26.2
- Service: `bigbench:local` (distroless, `GOMAXPROCS=0`)
- All tools ran against `127.0.0.1:8765` over loopback
- Duration: 15 s per endpoint (except Vegeta @ 500 rps open-loop;
  ab @ fixed 50 000 requests; Drill @ fixed 5 000 iterations × 3)
- Concurrency: 50 VUs / connections (Drill: 1, see notes below)

### 5.1 Throughput (requests/sec)

| Tool          | /simple (O(1)) | /medium (O(n log n)) | /heavy (O(2^n)) |
| ------------- | -------------: | -------------------: | --------------: |
| wrk           |     **184 991** |               51 003 |          33 214 |
| Bombardier    |         151 004 |               48 582 |          30 279 |
| hey           |         138 923 |               47 675 |          30 357 |
| k6 (mixed)    |    64 558 total |         (mixed run)  |   (mixed run)   |
| Siege (mixed) |     36 833 tps (across all 3 URLs round-robin) |   |        |
| ApacheBench   |          22 417 |               19 909 |          18 415 |
| Artillery     | 200 (arrivalRate-capped, open-loop) |          |                 |
| Vegeta        | 500 (rate-capped, open-loop)      |           |                 |
| Locust        |           2 306 |                1 547 |             761 |
| Drill         |     ~360 (single-threaded in this image, see §6) |    |      |

### 5.2 Latency (p95, where reported)

| Tool       | /simple p95 | /medium p95 | /heavy p95 |
| ---------- | ----------: | ----------: | ---------: |
| wrk        |      0.77 ms |      2.55 ms |     3.26 ms |
| Bombardier |      1.18 ms |      3.26 ms |     4.40 ms |
| hey        |      ~1.6 ms |      3.2 ms  |     4.2 ms  |
| k6 (all)   |      2.35 ms (aggregate p95)       |     |      |
| Vegeta     |      72 µs   |      199 µs  |     376 µs  |
| ab         |      3 ms    |      3 ms    |     3 ms    |
| Locust     |      12 ms   |      12 ms   |     12 ms   |

### 5.3 Reading the numbers

- **wrk, Bombardier, hey** sit at the top because they are
  closed-loop, multi-threaded, non-blocking Go/C clients with minimal
  per-request overhead. They accurately measure what the server can push.
- **k6 at 64.5 k/s aggregated** is the honest "mixed workload" number —
  k6 ran all three endpoints concurrently with 150 total VUs.
- **Vegeta and Artillery** are open-loop (rate-driven), so their RPS is
  whatever you *asked for*, not a ceiling. Their latencies (Vegeta p95 <
  400 µs even on `/heavy`) show the server is nowhere near saturated at
  500 rps.
- **ApacheBench** reports a lot of "Failed requests" on `/medium` and
  `/heavy`. These are **not real failures** — `ab` flags any response whose
  body length differs from the first response as "failed", and our JSON
  includes a per-request `elapsed_ms` that changes every time.
- **Locust @ ~4.6 k rps aggregated** is expected: it's CPU-bound by
  Python/gevent on a single process. The `--processes N` flag or a
  distributed worker setup would scale it linearly.
- **Drill 0.5.0 (xridge image)** serializes requests at concurrency=1 in
  this image version. The 358 rps number reflects serial loopback
  throughput, not drill's ceiling. A newer drill release supports
  `concurrency:` in YAML properly.

---

## 6. Caveats & known quirks

| Quirk                                      | Tool(s)         | Explanation                                                                                    |
| ------------------------------------------ | --------------- | ---------------------------------------------------------------------------------------------- |
| `Failed requests: N` on medium/heavy       | ApacheBench     | Response body length varies (dynamic `elapsed_ms`) — ab treats any length diff as "failed".     |
| High CPU warning in Locust logs            | Locust          | Python load generator is CPU-bound; add `--processes` or scale with workers for bigger runs.   |
| Concurrency=1 despite YAML                 | Drill (0.5.0)   | Old image; newer drill releases respect `concurrency:`. Bumping `iterations:` still works.     |
| `yokogawa/siege` image fails to pull       | Siege           | That image uses the deprecated v1 manifest; switched to `jstarcher/siege:latest`.               |
| `fcsonline/drill` image does not exist     | Drill           | Use `xridge/drill:latest` (publicly available), wired up in the Makefile.                       |

---

## 7. Reproducing

```bash
make build up        # start the app
make clean-results   # wipe old output
make test-all        # run every tool, write results/
make down            # cleanup
```

Raw output for every tool lives in `results/`. Each file is the full
stdout of the tool so you can reconstruct any metric not in the table
above (histograms, full percentiles, per-request logs).

---

## 8. Why this setup generalizes

The pattern here is deliberately minimal so it drops into other projects:

1. One Go service with controllable `O(1)`, `O(n log n)`, `O(2^n)`
   endpoints — lets you see whether the **tool** or the **service** is the
   bottleneck.
2. All tools containerized → zero host installs, reproducible, CI-ready.
3. All containers on `--network host` → no port plumbing, no bridge
   overhead.
4. A single Makefile target per tool + `test-all` for the full matrix.
5. Results captured as plain text in `results/`, so diffing runs across
   commits is trivial.
