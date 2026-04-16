## tjhop/prometheus-mcp-server

Go server. 28 tools — full Prometheus HTTP API plus embedded documentation.

### Run

```sh
docker compose up -d
# Prometheus UI: http://localhost:19092
# MCP (http):    http://localhost:18082
```

### Test over stdio

From `content/033/`:

```sh
python3 mcp_client.py \
  --call query '{"query":"up"}' \
  --call label_names '{}' \
  --call docs_list '{}' \
  -- docker run --rm -i --network mcp033-tjhop_obs \
       ghcr.io/tjhop/prometheus-mcp-server:latest \
       --prometheus.url=http://prometheus:9090 \
       --mcp.transport=stdio
```

Result: `initOk=true`, 28 tools, all 3 calls succeed. See `../results/tjhop.json`.

### Opting into dangerous admin tools

`delete_series`, `clean_tombstones`, and `snapshot` are hidden unless you pass `--dangerous.enable-tsdb-admin-tools` *and* run Prometheus with `--web.enable-admin-api`.

### Teardown

```sh
docker compose down -v
```
