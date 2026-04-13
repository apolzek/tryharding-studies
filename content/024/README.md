# GitHub Repo Pente-Fino Bot

A small bot that exposes an HTTP API where you register GitHub repositories and a
schedule. On the configured interval it fetches a detailed snapshot of each repo
straight from the GitHub API: languages, total size in bytes, stars, forks,
watchers, open/closed issues, open/closed PRs, contributors, license, topics and
timestamps. Results are stored in SQLite so you can inspect the history.

Stack: FastAPI + APScheduler + httpx + SQLite.

## Endpoints

| Method | Path                       | Description                              |
| ------ | -------------------------- | ---------------------------------------- |
| GET    | `/`                        | Health check                             |
| POST   | `/repos`                   | Register a repo and schedule it          |
| GET    | `/repos`                   | List registered repos                    |
| DELETE | `/repos/{id}`              | Unregister and stop the schedule         |
| POST   | `/repos/{id}/scan`         | Trigger a scan immediately               |
| GET    | `/repos/{id}/latest`       | Latest successful scan with metadata     |
| GET    | `/repos/{id}/scans?limit=N`| Scan history (success and failures)      |

`POST /repos` body:

```json
{ "owner": "fastapi", "repo": "fastapi", "interval_minutes": 60 }
```

`interval_minutes` defaults to `60`. A scan runs immediately on registration and
then repeats on the interval. Jobs are reloaded from the database at startup.

## Metadata collected

- `full_name`, `description`, `default_branch`, `license`, `topics`
- `main_language`, `languages_bytes`, `total_bytes`
- `stars`, `forks`, `watchers`
- `open_issues`, `closed_issues` (via Search API)
- `open_prs`, `closed_prs` (via Search API)
- `contributors` (anonymous included)
- `created_at`, `updated_at`, `pushed_at`, `archived`

## Running with Docker

```bash
docker build -t repo-pentefino .
docker run -d --name pentefino \
  -p 8000:8000 \
  -e GITHUB_TOKEN=ghp_your_token \
  -v $PWD/data:/app/data \
  repo-pentefino
```

`GITHUB_TOKEN` is optional but strongly recommended. Without it you are limited
to 60 REST requests/hour and 10 Search API requests/minute, which is tight once
you register more than a couple of repos.

## Running locally

```bash
pip install -r requirements.txt
uvicorn main:app --reload
```

## Quick test

```bash
# register a repo
curl -X POST http://localhost:8000/repos \
  -H 'Content-Type: application/json' \
  -d '{"owner":"fastapi","repo":"fastapi","interval_minutes":60}'

# fetch the latest snapshot
curl http://localhost:8000/repos/1/latest | jq
```

## Notes

- The GitHub Search API is used for issue/PR counts because the `/issues` and
  `/pulls` endpoints moved to cursor-based pagination and no longer expose a
  `rel="last"` link to count from.
- The canonical `full_name` returned by the repo endpoint is used when querying
  the Search API, so repo renames (e.g. `tiangolo/fastapi` → `fastapi/fastapi`)
  are handled transparently.
- SQLite file `bot.db` is created next to `main.py` at startup.
