## grafana/mcp-grafana

Official Grafana MCP server. 50 tools — by far the broadest surface: dashboards, folders, annotations, Prometheus/Loki/Pyroscope data-source queries, alerting, OnCall, Incident, Sift investigations, and panel rendering.

### Run

```sh
docker compose up -d
# Grafana: http://localhost:19095  (admin / admin)
# MCP (streamable-http): http://localhost:18085/mcp
```

A Prometheus data source is provisioned on boot (`uid=prometheus`).

### Test over stdio

From `content/033/`:

```sh
python3 mcp_client.py \
  --call list_datasources '{}' \
  --call search_dashboards '{"query":""}' \
  --call query_prometheus '{"datasourceUid":"prometheus","queryType":"instant","expr":"up"}' \
  --call list_prometheus_metric_names '{"datasourceUid":"prometheus","limit":5}' \
  -- docker run --rm -i --network mcp033-grafana_obs \
       -e GRAFANA_URL=http://grafana:3000 \
       -e GRAFANA_USERNAME=admin \
       -e GRAFANA_PASSWORD=admin \
       grafana/mcp-grafana:latest -t stdio
```

Result: `initOk=true`, 50 tools, all 4 calls succeed. See `../results/grafana.json`. The provisioned Prometheus data source is returned by `list_datasources`, and `list_prometheus_metric_names` returns real metric names scraped from the containerized Prometheus.

### Trimming the surface

Many of the 50 tools are gated behind Grafana features that may not be installed (OnCall, Incident, Sift, Pyroscope, image renderer). For production, pass flags such as `--disable-oncall --disable-incident --disable-pyroscope --disable-sift` or use `--enabled-tools <comma-separated>` to expose only what the agent needs.

Authentication in production should use a service-account token (`GRAFANA_SERVICE_ACCOUNT_TOKEN`), not the admin basic-auth used in this PoC.

### Teardown

```sh
docker compose down -v
```
