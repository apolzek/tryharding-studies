#!/usr/bin/env bash
# Drives each of the 5 MCP servers via stdio and asks it four RED-style
# questions about the demo-app. Each question is encoded as a tool/call with
# a specific PromQL query tailored to the server's argument schema.
#
# Assumes the OTel stack from docker-compose.yaml is already running and
# has accumulated metrics (run at least ~90 s after `docker compose up`).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CONTENT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
OUT_DIR="$SCRIPT_DIR/results"
NET=mcp033-otel-red_obs
mkdir -p "$OUT_DIR"

# Four RED-style PromQL queries against the spanmetrics output.
Q_RATE='sum by (span_name) (rate(traces_spanmetrics_calls_total{service_name="demo-app"}[1m]))'
Q_ERR='sum(rate(traces_spanmetrics_calls_total{service_name="demo-app",status_code="STATUS_CODE_ERROR"}[1m])) / sum(rate(traces_spanmetrics_calls_total{service_name="demo-app"}[1m]))'
Q_P95='histogram_quantile(0.95, sum by (le, span_name) (rate(traces_spanmetrics_duration_milliseconds_bucket{service_name="demo-app"}[1m])))'
Q_SLOWEST='topk(1, histogram_quantile(0.95, sum by (le, span_name) (rate(traces_spanmetrics_duration_milliseconds_bucket{service_name="demo-app"}[1m]))))'

json_q()  { jq -n --arg q "$1" '{query:$q}'; }
json_qg() { jq -n --arg e "$1" '{datasourceUid:"prometheus",queryType:"instant",expr:$e,endTime:"now"}'; }

echo "==> pab1it0"
python3 "$CONTENT_DIR/mcp_client.py" \
  --call execute_query "$(json_q "$Q_RATE")" \
  --call execute_query "$(json_q "$Q_ERR")" \
  --call execute_query "$(json_q "$Q_P95")" \
  --call execute_query "$(json_q "$Q_SLOWEST")" \
  -- docker run --rm -i --network "$NET" \
       -e PROMETHEUS_URL=http://prometheus:9090 \
       ghcr.io/pab1it0/prometheus-mcp-server:latest \
  > "$OUT_DIR/red-pab1it0.json" 2>/dev/null

echo "==> tjhop"
python3 "$CONTENT_DIR/mcp_client.py" \
  --call query "$(json_q "$Q_RATE")" \
  --call query "$(json_q "$Q_ERR")" \
  --call query "$(json_q "$Q_P95")" \
  --call query "$(json_q "$Q_SLOWEST")" \
  -- docker run --rm -i --network "$NET" \
       ghcr.io/tjhop/prometheus-mcp-server:latest \
       --prometheus.url=http://prometheus:9090 \
       --mcp.transport=stdio \
  > "$OUT_DIR/red-tjhop.json" 2>/dev/null

echo "==> giantswarm"
python3 "$CONTENT_DIR/mcp_client.py" \
  --call execute_query "$(json_q "$Q_RATE")" \
  --call execute_query "$(json_q "$Q_ERR")" \
  --call execute_query "$(json_q "$Q_P95")" \
  --call execute_query "$(json_q "$Q_SLOWEST")" \
  -- docker run --rm -i --network "$NET" \
       -e PROMETHEUS_URL=http://prometheus:9090 \
       local/mcp-prometheus:latest serve --transport=stdio \
  > "$OUT_DIR/red-giantswarm.json" 2>/dev/null

echo "==> vm"
python3 "$CONTENT_DIR/mcp_client.py" \
  --call query "$(json_q "$Q_RATE")" \
  --call query "$(json_q "$Q_ERR")" \
  --call query "$(json_q "$Q_P95")" \
  --call query "$(json_q "$Q_SLOWEST")" \
  -- docker run --rm -i --network "$NET" \
       -e VM_INSTANCE_ENTRYPOINT=http://victoriametrics:8428 \
       -e VM_INSTANCE_TYPE=single \
       ghcr.io/victoriametrics/mcp-victoriametrics:latest --mode=stdio \
  > "$OUT_DIR/red-vm.json" 2>/dev/null

echo "==> grafana"
python3 "$CONTENT_DIR/mcp_client.py" \
  --call query_prometheus "$(json_qg "$Q_RATE")" \
  --call query_prometheus "$(json_qg "$Q_ERR")" \
  --call query_prometheus "$(json_qg "$Q_P95")" \
  --call query_prometheus "$(json_qg "$Q_SLOWEST")" \
  -- docker run --rm -i --network "$NET" \
       -e GRAFANA_URL=http://grafana:3000 \
       -e GRAFANA_USERNAME=admin \
       -e GRAFANA_PASSWORD=admin \
       grafana/mcp-grafana:latest -t stdio \
  > "$OUT_DIR/red-grafana.json" 2>/dev/null

echo "done -> $OUT_DIR/red-*.json"
