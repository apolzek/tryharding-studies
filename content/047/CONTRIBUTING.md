# CONTRIBUTING.md — how to add a new POC to this batch

This is a drop-in template + the conventions all 11 existing POCs follow.
If you match this shape, a reader can clone the repo, `cd` into your
folder, run three commands, and reproduce your test.

## Before you start

1. Read the top-level [`README.md`](./README.md) for context.
2. Scan [`FINDINGS.md`](./FINDINGS.md) — the gotcha you're about to hit is
   probably already listed.
3. Check the target repo has a **published Docker image** (Docker Hub,
   GHCR, Quay, ECR). If it doesn't, either build from source inside the
   POC folder or add the project to [`skip.md`](./skip.md) with a reason.

## Folder layout

```
<tool-name>/
├── README.md            # required
├── docker-compose.yml   # or smoke.sh for pure CLI tools
├── test.sh              # optional — exercises the API/protocol
└── <config files>       # any YAML/conf bind-mounted into the container
```

- `tool-name` matches the upstream repo name, lowercased, dashes for slashes.
  (`AykutSarac/jsoncrack.com` → `jsoncrack/`.)
- README is mandatory. Compose or `smoke.sh` is mandatory. `test.sh`
  is "mandatory if the tool has an addressable API".

## Conventions

### Ports

- Allocate from `19001-19999`, bound to **`127.0.0.1`**, never `0.0.0.0`.
- Reserve the next free port by scanning [this README's port map](./README.md#port-map)
  and bumping by 1.
- Containers that don't need host access (private networks, e.g. the
  `otel-cli` ↔ `collector` pair) don't publish ports at all.

### Volumes

- Name volumes inside compose: `<tool>_data`, not `./data`.
- Always clean with `docker compose down -v` after the test — this keeps
  the POC stateless and makes re-running deterministic.
- If a tool writes to `/data`, set that explicitly; don't rely on the
  image's default.

### Read-only mounts

- Config files mount `:ro`.
- Docker socket mounts `:ro` (examples: `ctop`, `LoggiFly`).
- Never mount the host's `/etc`, `/root`, `$HOME`, or CWD writable.

### Image pinning

- Pin to a tag (`v0.25.0`, `1`, `:110.0`), **not** `:latest`, whenever the
  upstream publishes versioned tags. The only exception is CLI-one-shot
  tools where `:latest` is the only thing published.
- Record the exact tag in the POC README so re-runs are reproducible.

### Config files

- One config file per responsibility (e.g. `blackbox.yml`, `otel-collector.yaml`).
- Keep them minimal. A POC config should be the smallest diff from the
  upstream example that proves the feature you want to demo.

### Networking

- Use docker-compose's default bridge network per folder (the compose file
  owns its own `<tool>_default` network).
- When two containers in the same POC must talk, give the network a stable
  name (`name: my-tool-net`) so scripts like `send-span.sh` can
  `--network my-tool-net` to reach them.

### Test scripts

`test.sh` conventions:

- `set -euo pipefail`.
- Start with a readiness loop (`curl` in a `for`/`until` until `/healthz`
  responds, bounded to ~15 tries).
- Echo `==> <what you're about to do>` before each step.
- End with a clear pass/fail signal. Prefer `echo "OK: ..."` + `exit 0`
  vs. `echo "FAIL: ..." >&2 && exit 1`.

## README template

Drop this into `<tool-name>/README.md`, fill the blanks, delete any section
you don't need:

```markdown
# <tool-name>

Repo: https://github.com/<owner>/<repo>

<one-line description of what the tool does>

## What this POC tests

- <bullet the specific behavior being verified>
- <keep to 2–4 bullets>

## How to run

```bash
docker compose up -d
./test.sh
docker compose down -v
```

## What was verified

- <concrete observable outcome #1>
- <concrete observable outcome #2>

## Notes

<anything unusual: ports, config quirks, gotchas. Link to FINDINGS.md if the
  gotcha is interesting enough to record there.>

## Port

- Host: `127.0.0.1:<port>` → Container: `<container-port>`
```

## Checklist before calling a POC done

- [ ] `docker compose up -d` from a cold cache works (no pre-pulled images).
- [ ] `./test.sh` (or the manual smoke described in README) exits 0.
- [ ] `docker compose down -v` leaves **zero** leftover containers,
      volumes, or networks for this POC.
      Verify with `docker ps -a --filter name=<tool>-poc` (empty) and
      `docker volume ls | grep <tool>` (empty).
- [ ] README's *What was verified* is true, observable, and specific —
      not "it worked".
- [ ] Host port is bound to `127.0.0.1`, not `0.0.0.0`.
- [ ] Image is pinned to a versioned tag (except where only `:latest` exists).
- [ ] New row added to the [inventory table](./README.md#inventory-of-pocs)
      in the top-level README.
- [ ] New row added to the [port map](./README.md#port-map) if a host port
      was published.

## When to skip instead of adding

Put the repo in [`skip.md`](./skip.md) with a short reason if any of these
apply:

- No Docker image and building from source is > 15 min.
- Requires a GPU, hardware device, or account/credentials to run anything.
- TUI/desktop GUI where a meaningful smoke test needs a real TTY or X/Wayland
  session.
- Dual-use offensive tooling without a stated authorized engagement.
- Awesome-list, cheatsheet, or study material — no software to run.
- The URL isn't a repo (issue, gist, tree, blob, org page).

## Example: adding a hypothetical `toxiproxy` POC

```bash
mkdir toxiproxy
cat > toxiproxy/docker-compose.yml <<'YAML'
services:
  toxiproxy:
    image: ghcr.io/shopify/toxiproxy:2.11.0
    container_name: toxiproxy-poc
    ports:
      - "127.0.0.1:19009:8474"   # admin API
      - "127.0.0.1:19010:26379"  # a proxied upstream
    restart: unless-stopped
YAML

cat > toxiproxy/test.sh <<'SH'
#!/usr/bin/env bash
set -euo pipefail
BASE="http://127.0.0.1:19009"
# readiness, create a proxy, induce latency, assert, cleanup
SH
chmod +x toxiproxy/test.sh

# add to README.md inventory + port map
# add FINDINGS.md entry if anything surprised you
```

That's it. Commit, push, the next POC is 10 minutes of work.
