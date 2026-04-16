## pab1it0/prometheus-mcp-server

Python / FastMCP server. 6 tools, minimalist surface.

### Run

```sh
docker compose up -d
# Prometheus UI: http://localhost:19091
# MCP (http):    http://localhost:18081/mcp
```

### Test over stdio (recommended for CI / scripting)

From `content/033/`:

```sh
python3 mcp_client.py \
  --call list_metrics '{}' \
  --call execute_query '{"query":"up"}' \
  --call get_targets '{}' \
  -- docker run --rm -i --network mcp033-pab1it0_obs \
       -e PROMETHEUS_URL=http://prometheus:9090 \
       ghcr.io/pab1it0/prometheus-mcp-server:latest
```

Result: `initOk=true`, 6 tools, all 3 calls succeed. See `../results/pab1it0.json`.

### Teardown

```sh
docker compose down -v
```
