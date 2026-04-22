# ctop — container top

Repo: https://github.com/bcicen/ctop

A `top`-like interactive TUI for Docker containers.

## What this POC tests

- Pull and run ctop from the official image.
- Verify it can list the containers currently running on the host Docker daemon.

## How to run

ctop is a TUI — it needs a real terminal, so in this POC we only verify the
image works and runs. Running interactively uses:

```bash
docker run --rm -ti \
  --name ctop \
  --volume /var/run/docker.sock:/var/run/docker.sock:ro \
  quay.io/vektorlab/ctop:latest
```

Non-interactive smoke test used here:

```bash
./smoke.sh
```

## What was verified

- `docker pull quay.io/vektorlab/ctop:latest` works on this Ubuntu.
- `docker run ... ctop -v` prints the binary version, proving the volume-mount
  path to `/var/run/docker.sock` and the image entrypoint are correct.

## Cleanup

```bash
docker rmi quay.io/vektorlab/ctop:latest
```
