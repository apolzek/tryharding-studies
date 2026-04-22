# uptime-kuma

Repo: https://github.com/louislam/uptime-kuma

Self-hosted monitoring tool, like a lightweight UptimeRobot.

## What this POC tests

- Bring up uptime-kuma on `127.0.0.1:19001` (bound to loopback only, no public
  exposure).
- Confirm the web UI responds with HTTP 200 on the setup page.

## How to run

```bash
docker compose up -d
# first-boot init takes ~10s
curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:19001/
# open http://127.0.0.1:19001 in a browser to create the admin user
docker compose down -v   # destroys the volume as well
```

## What was verified

- Container `uptime-kuma-poc` came up healthy.
- `GET http://127.0.0.1:19001/` returns HTTP 200.
- The response body contains the Uptime Kuma HTML shell (`<title>Uptime Kuma</title>`).

## Port

- Host: `127.0.0.1:19001` → Container: `3001`
- Bound to loopback to avoid exposing admin setup on the LAN.
