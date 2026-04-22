#!/usr/bin/env bash
# Usage: sweep.sh <protocol: grpc|http> <cpus> <mem> [signal: traces|metrics|logs]
# Runs a stepped load test against the collector configured with the given
# resource limits, records per-step metrics into a CSV and prints the highest
# sustained rate that stayed <1% refused and kept the container alive.
set -euo pipefail

PROTO=${1:?protocol grpc|http}
CPUS=${2:?cpus e.g. 0.5, 1, 2}
MEM=${3:?memory e.g. 256m, 512m, 1g, 2g, 4g}
SIGNAL=${4:-traces}

STEP_DURATION=${STEP_DURATION:-30}
WORKERS_PER_GEN=${WORKERS_PER_GEN:-50}
PROM=${PROM:-http://localhost:9091}

ROOT=$(cd "$(dirname "$0")/.." && pwd)
OUT_DIR="$ROOT/results"
mkdir -p "$OUT_DIR"
RUN_CSV="$OUT_DIR/sweep_${PROTO}_${CPUS}cpu_${MEM}.csv"
BEST_CSV="$OUT_DIR/best.csv"

SERVICE="collector-$PROTO"
COMPOSE_FILE="$ROOT/otlp-$PROTO/docker-compose.yml"
IMG_GEN="ghcr.io/open-telemetry/opentelemetry-collector-contrib/telemetrygen:v0.150.0"

# Rate plan per memory tier (total signals/s). Step grows until saturation or OOM.
case "$MEM" in
  256m) TOTAL_RATES="5000 15000 30000 60000 120000 200000 300000" ;;
  512m) TOTAL_RATES="10000 30000 60000 120000 200000 300000 450000 600000" ;;
  1g)   TOTAL_RATES="20000 60000 120000 240000 360000 500000 700000 900000" ;;
  2g)   TOTAL_RATES="50000 120000 240000 400000 600000 900000 1200000 1600000" ;;
  4g)   TOTAL_RATES="100000 240000 500000 800000 1200000 1600000 2000000 2500000" ;;
  *)    TOTAL_RATES="20000 60000 120000 240000 500000" ;;
esac

case "$PROTO" in
  grpc) GEN_FLAGS="--otlp-endpoint=$SERVICE:4317 --otlp-insecure" ;;
  http) GEN_FLAGS="--otlp-endpoint=$SERVICE:4318 --otlp-http --otlp-insecure" ;;
  *)    echo "invalid protocol $PROTO" >&2; exit 1 ;;
esac

# Realistic-ish spans: 8 attributes (adds ~500B per span)
GEN_ATTRS=(
  --telemetry-attributes 'service.name="api-gateway"'
  --telemetry-attributes 'service.version="1.42.3"'
  --telemetry-attributes 'deployment.environment="production"'
  --telemetry-attributes 'k8s.pod.name="api-5fd9c8b7d6-xj4l2"'
  --telemetry-attributes 'http.method="POST"'
  --telemetry-attributes 'http.route="/api/v1/orders"'
  --telemetry-attributes 'http.status_code=200'
  --telemetry-attributes 'trace.sampler="parent_based"'
)

echo "=========================================================="
echo "Sweep: proto=$PROTO cpus=$CPUS mem=$MEM signal=$SIGNAL"
echo "Rates: $TOTAL_RATES"
echo "=========================================================="

# Recreate collector with the requested limits
docker compose -f "$COMPOSE_FILE" down --remove-orphans >/dev/null 2>&1 || true
COLLECTOR_CPUS=$CPUS COLLECTOR_MEM=$MEM \
  docker compose -f "$COMPOSE_FILE" up -d >/dev/null

case "$PROTO" in
  grpc) HEALTH_URL="http://localhost:13133/" ;;
  http) HEALTH_URL="http://localhost:13134/" ;;
esac
for i in $(seq 1 30); do
  if curl -sf "$HEALTH_URL" >/dev/null 2>&1; then break; fi
  sleep 1
done
curl -sf "$HEALTH_URL" >/dev/null || {
  echo "collector never healthy"; docker logs --tail 40 "$SERVICE"; exit 1
}
echo "collector $SERVICE healthy — $(docker inspect -f '{{.HostConfig.NanoCpus}}ncpu / {{.HostConfig.Memory}}B mem' "$SERVICE")"

# Write CSV header (overwrite per run)
echo "protocol,cpus,mem,signal,target_rate,generators,workers_each,per_worker_rate,received_rate,refused_rate,cpu_cores,cpu_util_pct,mem_rss_bytes,container_alive" > "$RUN_CSV"

# Prometheus query helper → prints float value (0 if empty)
prom_q() {
  local q=$1; local at=$2
  curl -sG "$PROM/api/v1/query" --data-urlencode "query=$q" --data-urlencode "time=$at" \
    | python3 -c "import json,sys;
d=json.load(sys.stdin).get('data',{}).get('result',[])
print(float(d[0]['value'][1]) if d else 0.0)" 2>/dev/null || echo "0.0"
}

# Spawn N telemetrygen containers in parallel. Each runs for $STEP_DURATION seconds
# with the given rate. Returns when all have exited.
run_load() {
  local total_rate=$1
  local duration=$2
  # Scale generators: single telemetrygen ceiling ~150k/s, so spawn 1 gen per 100k target
  local n_gens
  if   [ "$total_rate" -le 100000 ]; then n_gens=1
  elif [ "$total_rate" -le 200000 ]; then n_gens=2
  elif [ "$total_rate" -le 400000 ]; then n_gens=4
  elif [ "$total_rate" -le 800000 ]; then n_gens=6
  elif [ "$total_rate" -le 1500000 ]; then n_gens=10
  else n_gens=16
  fi
  local rate_each=$(( total_rate / n_gens ))
  # Keep per_worker_rate ≤ 5000 (more workers scale better than higher rate per worker)
  local workers=$WORKERS_PER_GEN
  local per_worker=$(( rate_each / workers ))
  if [ "$per_worker" -lt 1 ]; then per_worker=1; workers=$rate_each; fi

  # Output for logging
  echo "$n_gens $workers $per_worker"

  local pids=()
  for i in $(seq 1 "$n_gens"); do
    docker run --rm -d --network otelbench --name "tg-$i-$$" \
      "$IMG_GEN" "$SIGNAL" \
      --workers "$workers" \
      --rate "$per_worker" \
      --duration "${duration}s" \
      "${GEN_ATTRS[@]}" \
      $GEN_FLAGS >/dev/null
    pids+=("tg-$i-$$")
  done

  # Wait for all to finish
  for p in "${pids[@]}"; do
    docker wait "$p" >/dev/null 2>&1 || true
    docker rm -f "$p" >/dev/null 2>&1 || true
  done
}

signal_metric="otelcol_receiver_accepted_spans_total"
refused_metric="otelcol_receiver_refused_spans_total"
case "$SIGNAL" in
  metrics) signal_metric="otelcol_receiver_accepted_metric_points_total"
           refused_metric="otelcol_receiver_refused_metric_points_total" ;;
  logs)    signal_metric="otelcol_receiver_accepted_log_records_total"
           refused_metric="otelcol_receiver_refused_log_records_total" ;;
esac

CPU_LIMIT=$(python3 -c "print(float('$CPUS'))")

best_rate=0
for target in $TOTAL_RATES; do
  echo ""
  echo ">>> target=${target} ${SIGNAL}/s  duration=${STEP_DURATION}s"

  sleep 3  # gap so prior window doesn't bleed in
  t_start=$(date +%s)
  read -r n_gens workers per_worker <<<"$(run_load "$target" "$STEP_DURATION")"
  t_end=$(date +%s)
  elapsed=$((t_end - t_start))
  sleep 4  # let prometheus scrape tail

  # Measurement window: skip first 8s ramp-up from window end
  meas_window=$((elapsed - 8))
  if [ "$meas_window" -lt 15 ]; then meas_window=$elapsed; fi
  query_ts=$((t_end - 2))

  received_rate=$(prom_q "sum(rate(${signal_metric}{protocol=\"$PROTO\"}[${meas_window}s]))" $query_ts)
  refused_rate=$(prom_q "sum(rate(${refused_metric}{protocol=\"$PROTO\"}[${meas_window}s]))" $query_ts)
  cpu=$(prom_q "rate(otelcol_process_cpu_seconds_total{protocol=\"$PROTO\"}[${meas_window}s])" $query_ts)
  mem_rss=$(prom_q "max_over_time(otelcol_process_memory_rss_bytes{protocol=\"$PROTO\"}[${elapsed}s])" $query_ts)

  alive="true"
  if ! docker inspect -f '{{.State.Running}}' "$SERVICE" 2>/dev/null | grep -q true; then
    alive="false"
  fi
  cpu_util_pct=$(python3 -c "print(round($cpu / $CPU_LIMIT * 100, 1))")

  printf "    gens=%d wrk=%d per_wrk=%d → received=%.0f refused=%.0f cpu=%.2f(%s%% of %sC) mem=%.0f alive=%s\n" \
    "$n_gens" "$workers" "$per_worker" \
    "$received_rate" "$refused_rate" "$cpu" "$cpu_util_pct" "$CPUS" "$mem_rss" "$alive"

  printf "%s,%s,%s,%s,%d,%d,%d,%d,%.0f,%.0f,%.3f,%.1f,%.0f,%s\n" \
    "$PROTO" "$CPUS" "$MEM" "$SIGNAL" "$target" "$n_gens" "$workers" "$per_worker" \
    "$received_rate" "$refused_rate" "$cpu" "$cpu_util_pct" "$mem_rss" "$alive" >> "$RUN_CSV"

  if [ "$alive" = "false" ]; then
    echo "!!! collector died (likely OOM-killed) — sweep aborted at target=$target"
    break
  fi

  # Saturation criterion: received < 85% of target OR refused > 1% of target
  ok=$(python3 -c "print(1 if $received_rate >= 0.85 * $target and $refused_rate < max(10, 0.01 * $target) else 0)")
  if [ "$ok" = "1" ]; then
    best_rate=$(python3 -c "print(int($received_rate))")
  else
    echo "--- saturation: accepted=$(printf %.0f $received_rate) < 85% of target=$target OR refused>=1%"
    break
  fi
done

echo ""
echo "=========================================================="
echo "BEST sustained for $PROTO $CPUS cpu $MEM: $best_rate ${SIGNAL}/s"
echo "=========================================================="
echo "$PROTO,$CPUS,$MEM,$SIGNAL,$best_rate" >> "$BEST_CSV"

docker compose -f "$COMPOSE_FILE" down --remove-orphans >/dev/null 2>&1 || true
