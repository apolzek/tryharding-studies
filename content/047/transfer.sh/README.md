# transfer.sh

Repo: https://github.com/dutchcoders/transfer.sh

Self-hosted file sharing service. `curl` upload, get a link, share/download.

## What this POC tests

- Boot transfer.sh with `local` provider on `127.0.0.1:19003`.
- Upload a file via PUT, re-download it, and check the content matches.

## How to run

```bash
docker compose up -d
./test.sh
docker compose down -v
```

## What was verified

- `PUT /<name>` uploads the file and returns a download URL.
- `GET <url>` returns the original bytes (sha256 match).
