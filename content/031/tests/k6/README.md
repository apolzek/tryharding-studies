# k6 load tests

Three scenarios run concurrently for 1m:

- `browse`:   20 VUs -> `GET /api/feed`
- `interact`: 10 VUs -> `POST /api/interactions`
- `replay`:   5  VUs -> `POST /replay/ingest`

Thresholds: p95 < 500ms, error rate < 1%.

## Run (inside the obs-net network, service name resolves)

```bash
docker run --rm -i --network obs-net grafana/k6 run - < load.js
```

## Run from host

```bash
docker run --rm -i --network host -e BASE_URL=http://localhost:8080 grafana/k6 run - < load.js
```

## Or with local k6

```bash
BASE_URL=http://localhost:8080 k6 run load.js
```
