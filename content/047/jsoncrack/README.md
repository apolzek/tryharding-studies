# jsoncrack

Repo: https://github.com/AykutSarac/jsoncrack.com

Web app that visualizes JSON as an interactive graph. Useful for quickly
inspecting nested API responses, manifests, config files.

## What this POC tests

- Boot a self-hosted build on `127.0.0.1:19008`.
- Confirm the HTML shell is served with a `200 OK` and the expected `<title>`.

## How to run

```bash
docker compose up -d
curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:19008/
curl -s http://127.0.0.1:19008/ | grep -oE '<title>[^<]+</title>' | head -1
docker compose down -v
```

## What was verified

- Static app served HTTP 200 on root path.
- HTML `<title>` contains "JSON Crack".

## Notes

- Upstream doesn't publish an official Docker image. Community image
  `gorkinus/jsoncrack:latest` was used here and it happens to listen on `8080`
  internally, not the `8888` seen in some docs. For production, build from
  source (`docker build .` against `AykutSarac/jsoncrack.com`) and pin a
  specific commit.
