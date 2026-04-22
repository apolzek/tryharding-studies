# LoggiFly

Repo: https://github.com/clemcer/LoggiFly

Headless log-tailer for Docker containers. Watches container logs, matches
keywords/regex, and fires notifications (ntfy, Apprise, etc).

## What this POC tests

- Boot LoggiFly with a random `ntfy.sh` topic (so misconfig won't spam anyone).
- Confirm it attaches to the Docker socket and begins monitoring all running
  containers for the keyword `critical`.
- Spawn a throwaway container that logs `critical: poc-047` and check
  LoggiFly detects the keyword.

## How to run

```bash
docker compose up -d
./test.sh
docker compose down -v
```

## What was verified

- `docker logs loggifly-poc` shows LoggiFly loading config and listing
  monitored containers.
- After a test container logs `critical: poc-047`, LoggiFly reports a keyword
  match in its own log output.

## Notes

- No HTTP endpoint — verification is 100% by container logs.
- Since the ntfy topic is random, you don't have to worry about spurious
  notifications going anywhere meaningful.
