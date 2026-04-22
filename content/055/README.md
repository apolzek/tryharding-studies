# 055 — Stalkr: handle-based "spy" SaaS (no paid APIs)

A small SaaS that continuously watches a social handle across multiple
public-facing networks. The client adds an `@handle`, picks which networks to
monitor, and the backend scrapes (HTML + public RSS/Atom feeds) on a schedule;
the UI then surfaces a live feed and a daily digest.

**No third-party API tokens are required.** Every collector uses public HTML
pages or the public RSS/Atom endpoints that the networks themselves publish.

## Networks supported out of the box

| Network    | Source used                                     | Notes                         |
|------------|--------------------------------------------------|-------------------------------|
| GitHub     | `github.com/{user}.atom` + profile HTML          | activity + daily profile snap |
| Dev.to     | `dev.to/feed/{user}`                             | RSS                           |
| Medium     | `medium.com/feed/@{user}`                        | RSS                           |
| Reddit     | `old.reddit.com/user/{user}/.rss`                | public Atom, no login         |
| YouTube    | resolves `@handle` → channel_id → channel Atom    | zero auth                     |
| Hacker News| `news.ycombinator.com/submitted?id={user}` HTML  | scraped                       |
| Mastodon   | `https://{instance}/@{user}.rss`                 | pass `user@instance` to pick  |
| X/Twitter  | `{NITTER_BASE}/{user}/rss` via a Nitter instance | no API key                    |

## Architecture

```
┌──────────────┐   polling (15m default)   ┌──────────────────┐
│ APScheduler  │ ────────────────────────► │  scrapers/*.py   │
└──────────────┘                            └──────────────────┘
        │                                            │
        ▼                                            ▼
   SQLite (events, targets, runs)  ◄── FastAPI ──► React UI (Vite)
```

- **Backend** — FastAPI + SQLAlchemy + APScheduler + httpx/BeautifulSoup/feedparser.
- **Frontend** — React + Vite, lightweight "spy" dark theme.
- **Persistence** — SQLite on a named docker volume (`stalkr-data`).

## Running it

```bash
cd content/055
docker compose up --build
# UI:   http://localhost:5173
# API:  http://localhost:8000/docs
```

The backend scrapes every `SCRAPE_INTERVAL_MINUTES` (default 15) and also runs
once on startup. You can trigger an on-demand scrape per target via the UI's
"scrape" button or `POST /api/targets/{id}/scrape`.

## Testing with @apolzek

Once `docker compose up` is running, open http://localhost:5173 and add
`@apolzek` with any subset of networks ticked (GitHub is the one guaranteed to
produce events for that handle). Then click **scrape** on the target card to
force an immediate pull — the feed and daily digest will populate within a few
seconds.

Equivalent CLI sanity check:

```bash
curl -X POST localhost:8000/api/targets \
  -H 'content-type: application/json' \
  -d '{"handle":"apolzek","networks":["github","devto","reddit","hackernews"]}'
curl -X POST localhost:8000/api/targets/1/scrape | jq
curl localhost:8000/api/targets/1/digest?hours=24 | jq
```

## Running tests

```bash
cd backend
pip install -r requirements.txt
pytest
```

The tests feed fixture payloads through the scrapers (httpx is monkey-patched)
so they pass offline.

## Environment variables

| Var                       | Default              | Purpose                          |
|---------------------------|----------------------|----------------------------------|
| `SCRAPE_INTERVAL_MINUTES` | `15`                 | Scheduler interval               |
| `HTTP_TIMEOUT_SECONDS`    | `20`                 | Per-request timeout              |
| `NITTER_BASE`             | `https://nitter.net` | Nitter instance used for twitter |
| `MASTODON_INSTANCE`       | `mastodon.social`    | Default when handle has no `@server`|
| `DATA_DIR`                | `/data`              | Where SQLite is written          |

## Notes on scraping etiquette

- A descriptive `User-Agent` is sent on every request.
- The default interval (15 min) is intentionally conservative.
- If a source blocks an instance (e.g. a particular Nitter server), swap
  `NITTER_BASE` or disable that network on the target.
- Public handle-observability only — nothing here authenticates, logs in, or
  accesses non-public content.
