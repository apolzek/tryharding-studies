---
title: GitHub repository inspection bot with scheduled metadata snapshots
tags: [automation, python, github-api, telegram-bot, sqlite]
status: stable
---

## GitHub repository inspection bot with scheduled metadata snapshots

### Objectives

Build a small bot that exposes an HTTP API where you register GitHub repositories and a schedule. On the configured interval it fetches a detailed snapshot of each repo straight from the GitHub API — languages, total size in bytes, stars, forks, watchers, open/closed issues, open/closed PRs, contributors, license, topics and timestamps — and stores every run in SQLite so history is queryable. The PoC exercises rate-limit-aware GitHub API use (Search API for issue/PR counts, graceful handling of unauthenticated quotas), APScheduler job persistence across restarts, and a clean split between scheduling and metadata fetching.

Stack: FastAPI + APScheduler + httpx + SQLite.

### Prerequisites

- Docker (or Python 3.11+ with `pip`)
- A GitHub personal access token (optional but strongly recommended — the anonymous quota is 60 REST requests/hour and 10 Search API requests/minute)

### Reproducing

Run with Docker:

```bash
docker build -t repo-pentefino .
docker run -d --name pentefino \
  -p 8000:8000 \
  -e GITHUB_TOKEN=ghp_your_token \
  -v $PWD/data:/app/data \
  repo-pentefino
```

Run locally:

```bash
pip install -r requirements.txt
uvicorn main:app --reload
```

Register a repository and fetch the latest snapshot:

```bash
# Register and schedule
curl -X POST http://localhost:8000/repos \
  -H 'Content-Type: application/json' \
  -d '{"owner":"fastapi","repo":"fastapi","interval_minutes":60}'

# Latest successful snapshot
curl http://localhost:8000/repos/1/latest | jq
```

Endpoints:

| Method | Path | Description |
|---|---|---|
| GET | `/` | Health check |
| POST | `/repos` | Register a repo and schedule it |
| GET | `/repos` | List registered repos |
| DELETE | `/repos/{id}` | Unregister and stop the schedule |
| POST | `/repos/{id}/scan` | Trigger a scan immediately |
| GET | `/repos/{id}/latest` | Latest successful scan with metadata |
| GET | `/repos/{id}/scans?limit=N` | Scan history (success and failures) |

`interval_minutes` defaults to `60`. A scan runs immediately on registration and then repeats on the interval. Jobs are reloaded from the database at startup.

Metadata collected on each scan:

- `full_name`, `description`, `default_branch`, `license`, `topics`
- `main_language`, `languages_bytes`, `total_bytes`
- `stars`, `forks`, `watchers`
- `open_issues`, `closed_issues` (via Search API)
- `open_prs`, `closed_prs` (via Search API)
- `contributors` (anonymous included)
- `created_at`, `updated_at`, `pushed_at`, `archived`

### Results

The GitHub Search API turned out to be the right tool for issue/PR counts because the `/issues` and `/pulls` endpoints moved to cursor-based pagination and no longer expose a `rel="last"` link you can count from. Using the canonical `full_name` returned by the repo endpoint when querying the Search API handles repo renames transparently (e.g. `tiangolo/fastapi` → `fastapi/fastapi`). APScheduler's job store on top of SQLite means schedules survive restarts without extra code, and keeping both successful and failed scans in history makes it easy to correlate later rate-limit bursts with missing data. An authenticated token is effectively mandatory once more than a couple of repos are registered.

### References

```
🔗 https://docs.github.com/en/rest
🔗 https://docs.github.com/en/rest/search/search
🔗 https://fastapi.tiangolo.com/
🔗 https://apscheduler.readthedocs.io/en/stable/
🔗 https://www.python-httpx.org/
```
