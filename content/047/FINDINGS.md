# FINDINGS.md — gotchas discovered running the 11 POCs

The stuff I had to learn the hard way while bringing these tools up. Every
entry is something that caused me to `docker logs` or re-read docs. Keeping
this file so the next person (or me, in three months) gets the answer in
seconds instead of minutes.

## 1. `otel-cli`: gRPC endpoint needs `--insecure` explicitly

**Symptom:** `exec ... -- echo hi` returns success, but the collector never
logs the span.

**Cause:** `otel-cli` defaults to `insecure: false`. Without `--insecure`
(or `OTEL_EXPORTER_OTLP_INSECURE=true`) it tries TLS against a plaintext
port and silently fails.

**Fix:** always pass `--endpoint host:4317 --protocol grpc --insecure` (or
set env vars) when targeting a dev/test collector. Also: don't rely on
`OTEL_EXPORTER_OTLP_ENDPOINT` starting with `http://` — that doesn't imply
insecure the way it does for some other OTEL SDKs.

**Where it bit me:** [`otel-cli/send-span.sh`](./otel-cli/send-span.sh).

## 2. `otel-tui`: entrypoint is `/otel-tui`, not shell-style argv

**Symptom:** `docker run ymtdzzz/otel-tui:latest --help` →
`exec: "--help": executable file not found in $PATH`.

**Cause:** the image has `Entrypoint: []` and `Cmd: ["/otel-tui"]`.
Appending `--help` replaces the CMD, so Docker tries to exec `--help`.

**Fix:** either
```bash
docker run --rm --entrypoint /otel-tui ymtdzzz/otel-tui:latest --help
# or
docker run --rm ymtdzzz/otel-tui:latest /otel-tui --help
```

**Where it bit me:** [`otel-tui/smoke.sh`](./otel-tui/smoke.sh).

## 3. `endlessh-go`: flag is `-enable_prometheus`, not `-prometheus_enabled`

**Symptom:** container boot-loops; `docker logs` shows
`flag provided but not defined: -prometheus_enabled`.

**Cause:** docs in third-party posts drift. The actual flag in the binary
(confirmed by `./endlessh --help`) is `-enable_prometheus`. The companion
flags `-prometheus_host`, `-prometheus_port`, `-prometheus_entry` *are*
real; only the enabler has a different name.

**Fix:** use `-enable_prometheus` (no value — it's a boolean). Full working
command in [`endlessh-go/docker-compose.yml`](./endlessh-go/docker-compose.yml).

## 4. `jsoncrack`: no official image, community image listens on 8080

**Symptom:** following docs that say `-p 8888:8888` leaves `curl` returning
`HTTP 000` / connection refused.

**Cause:** upstream `AykutSarac/jsoncrack.com` doesn't publish an official
Docker image. The community image I used (`gorkinus/jsoncrack:latest`)
serves nginx on **port 8080**, not 8888. Blog posts reference 8888 because
that was the local dev port in an earlier Next.js setup.

**Fix:** always `docker exec <name> cat /etc/nginx/conf.d/default.conf`
(or `ss -tln`) when the documented port doesn't respond. The authoritative
answer is what the container actually binds.

**Where it bit me:** [`jsoncrack/docker-compose.yml`](./jsoncrack/docker-compose.yml).

## 5. `LoggiFly`: `GLOBAL_KEYWORDS` alone is not enough

**Symptom:** LoggiFly boots, reports keywords, and says
`❌ No containers or Swarm services are configured.` even though there are
running containers to monitor.

**Cause:** by default LoggiFly only watches containers declared explicitly
(`CONTAINERS=...`). Global keywords define *what* to match, not *where*.

**Fix:** set `MONITOR_ALL_CONTAINERS=true`. Combined with `GLOBAL_KEYWORDS`,
it tails every running container and fires on keyword match.

**Bonus gotcha:** LoggiFly doesn't attach to containers that exit in under
~1s. If you're using a throwaway container to prove the keyword match,
keep it alive long enough (a `sleep` tail works) or LoggiFly misses the
attach window.

**Where it bit me:** [`loggifly/docker-compose.yml`](./loggifly/docker-compose.yml)
and [`loggifly/test.sh`](./loggifly/test.sh).

## 6. `transfer.sh`: `local` provider needs `--basedir` and a volume

**Symptom:** upload returns a URL, but re-downloading it 404s.

**Cause:** without `--basedir /data` + a mounted volume, `local` provider
writes to `/tmp/transfer.sh` inside the container, which can be reaped.

**Fix:** explicit `--basedir /data` with a named volume, then
`docker compose down -v` to clean. See
[`transfer.sh/docker-compose.yml`](./transfer.sh/docker-compose.yml).

## 7. `blackbox_exporter`: config must be mounted, not baked in

**Symptom:** `/probe?module=http_2xx&target=...` returns
`Unknown module "http_2xx"`.

**Cause:** the default image ships with `blackbox.yml` that may not define
the module you need (or you want custom probe timeouts). The image expects
`--config.file=/etc/blackbox/blackbox.yml`; you must mount your own.

**Fix:** ship `blackbox.yml` in the POC folder and bind-mount it read-only.
Example in [`blackbox_exporter/`](./blackbox_exporter/).

## 8. `uptime-kuma`: redirects `/ → /dashboard`

**Symptom:** `curl -sI http://127.0.0.1:19001/` returns `302 Found`, which
looks like a failure if you were expecting 200.

**Cause:** first-boot of Uptime Kuma redirects the root URL to either
`/setup` (fresh install) or `/dashboard` (already set up). 302 is the
correct response.

**Fix:** probe with `curl -L` to follow, or directly probe `/dashboard`.

## 9. Bookmarks list includes "not a project" URLs

`repositories.yaml` mixes real repos with:

- bare org pages (`orgs/nochaosio/...`)
- gists (`danielkec/...`, `glaucia86/...`)
- deep paths (`.../issues/889`, `.../blob/master/docker-compose.yml`)
- generic landing pages (`https://github.com/`, `https://codeberg.org/`)

Any script that enumerates "all bookmarked repos" needs to strip these
first (`awk -F'/' '{print $4"/"$5}'` + filter against `issues|blob|tree|orgs`
+ dedupe).

## 10. Running containers reserved ports 11xxx and 16xxx, 18xxx

Before assigning POC ports I checked `docker ps --format '{{.Ports}}'`.
The `rag-*`, `rrk-*`, `hyb-*`, `vec-*`, `kw-*` stacks reserved
`11434-11435`, `16333-16336`, `18001-18005`. I used `19001-19008` so the
POC batch and the existing stacks never fight for ports. Future POCs
should stay in `19000-19999` unless there's a good reason.
