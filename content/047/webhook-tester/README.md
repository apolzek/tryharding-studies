# webhook-tester

Repo: https://github.com/tarampampam/webhook-tester

Self-hosted clone of webhook.site — lets you create a unique URL, POST to it,
and inspect the payload.

## What this POC tests

- Boot webhook-tester on `127.0.0.1:19002`.
- Create a session via the API, POST to the capture URL, then list captured
  requests and confirm the body/headers round-trip correctly.

## How to run

```bash
docker compose up -d
./test.sh
docker compose down -v
```

## What was verified

- `POST /api/session` creates a session and returns a UUID.
- `POST /<uuid>` captures request body and headers.
- `GET /api/session/<uuid>/requests` lists captured requests including the JSON
  payload we sent.
