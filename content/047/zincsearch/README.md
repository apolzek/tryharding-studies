# zincsearch

Repo: https://github.com/zincsearch/zincsearch

Lightweight full-text search server — Elasticsearch-ish API, Go, single
binary, low RAM footprint.

## What this POC tests

- Boot zincsearch on `127.0.0.1:19004` with `admin/Admin123!`.
- Create an index, bulk-insert a few docs, run a search query, verify hits.

## How to run

```bash
docker compose up -d
./test.sh
docker compose down -v
```

## What was verified

- `GET /version` returns version info.
- `POST /api/_bulkv2` ingests 3 docs into index `poc047`.
- `POST /api/poc047/_search` returns a hit for query `rinha`.
