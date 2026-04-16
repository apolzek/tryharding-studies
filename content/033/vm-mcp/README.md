## VictoriaMetrics/mcp-victoriametrics

Official VictoriaMetrics MCP server. 16 tools — Prometheus-compatible queries plus VM-specific tools (`active_queries`, `top_queries`, `metric_statistics`, `tsdb_status`, `explain_query`, `prettify_query`) and embedded VM documentation.

### Run

```sh
docker compose up -d
# VictoriaMetrics: http://localhost:18428
# MCP (http):      http://localhost:18084/mcp
```

`vmagent` scrapes `victoriametrics` itself and `node-exporter` and remote-writes to `victoriametrics:8428`, so the backend has real data after ~5 s.

### Test over stdio

From `content/033/`:

```sh
python3 mcp_client.py \
  --call query '{"query":"up"}' \
  --call metrics '{}' \
  --call labels '{}' \
  --call tsdb_status '{}' \
  -- docker run --rm -i --network mcp033-vm_obs \
       -e VM_INSTANCE_ENTRYPOINT=http://victoriametrics:8428 \
       -e VM_INSTANCE_TYPE=single \
       ghcr.io/victoriametrics/mcp-victoriametrics:latest --mode=stdio
```

Result: `initOk=true`, 16 tools, all 4 calls succeed. See `../results/vm.json`.

### VictoriaMetrics Cloud

For VM Cloud, set `VMC_API_KEY` instead of `VM_INSTANCE_ENTRYPOINT`. VM cluster mode uses `VM_INSTANCE_TYPE=cluster` and requires the `select` / `insert` URLs.

### Teardown

```sh
docker compose down -v
```
