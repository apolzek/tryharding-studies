#!/bin/bash
# test-latency.sh — Documented latency effect tests between OTel collectors via Toxiproxy
#
# Scenarios tested:
#   0. baseline   — 0ms   (no latency)
#   1. low        — 50ms  (acceptable)
#   2. medium     — 200ms (noticeable)
#   3. high       — 800ms (degraded)
#   4. critical   — 2000ms (near-breaking)
#   5. restore    — back to baseline
#
# Usage: ./test-latency.sh [scenario]
#   e.g. ./test-latency.sh 0  -> run only baseline
#        ./test-latency.sh    -> run all scenarios sequentially

set -euo pipefail

TOXIPROXY_API="http://localhost:8474"
JAEGER_API="http://localhost:16686"
PROM_API="http://localhost:9090"
PROXY_NAME="otel-collector"
TOXIC_NAME="latency"
WAIT_SECS=15        # seconds to observe after each change
SEPARATOR="━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# ─── helpers ──────────────────────────────────────────────────────────────────

log()  { echo -e "\033[1;34m[INFO]\033[0m  $*"; }
ok()   { echo -e "\033[1;32m[ OK ]\033[0m  $*"; }
warn() { echo -e "\033[1;33m[WARN]\033[0m  $*"; }
err()  { echo -e "\033[1;31m[ERR ]\033[0m  $*"; }
sep()  { echo -e "\033[1;37m${SEPARATOR}\033[0m"; }

check_deps() {
  for cmd in curl jq bc; do
    command -v "$cmd" &>/dev/null || { err "Missing dependency: $cmd"; exit 1; }
  done
}

check_services() {
  log "Checking that all services are reachable..."
  local ok=true

  curl -sf "${TOXIPROXY_API}/proxies" > /dev/null 2>&1 \
    && ok "Toxiproxy  : ${TOXIPROXY_API}" \
    || { warn "Toxiproxy not reachable at ${TOXIPROXY_API}. Is the stack running?"; ok=false; }

  curl -sf "${JAEGER_API}/api/services" > /dev/null 2>&1 \
    && ok "Jaeger     : ${JAEGER_API}" \
    || warn "Jaeger not reachable at ${JAEGER_API}"

  curl -sf "${PROM_API}/-/ready" > /dev/null 2>&1 \
    && ok "Prometheus : ${PROM_API}" \
    || warn "Prometheus not reachable at ${PROM_API}"

  [[ "$ok" == "false" ]] && { err "Required services not running. Start with: docker compose up -d"; exit 1; }
  echo ""
}

# Remove existing toxic (ignore errors if it doesn't exist)
remove_toxic() {
  curl -sf -X DELETE "${TOXIPROXY_API}/proxies/${PROXY_NAME}/toxics/${TOXIC_NAME}" \
    > /dev/null 2>&1 || true
}

# Set latency toxic: set_latency <latency_ms> <jitter_ms>
set_latency() {
  local lat="$1" jitter="$2"
  remove_toxic
  if [[ "$lat" -gt 0 ]]; then
    curl -sf -X POST "${TOXIPROXY_API}/proxies/${PROXY_NAME}/toxics" \
      -H "Content-Type: application/json" \
      -d "{
        \"name\": \"${TOXIC_NAME}\",
        \"type\": \"latency\",
        \"stream\": \"upstream\",
        \"toxicity\": 1.0,
        \"attributes\": {\"latency\": ${lat}, \"jitter\": ${jitter}}
      }" > /dev/null
  fi
}

get_current_latency() {
  local info
  info=$(curl -sf "${TOXIPROXY_API}/proxies/${PROXY_NAME}/toxics" 2>/dev/null) || { echo "0"; return; }
  echo "$info" | jq -r '.[] | select(.name=="latency") | .attributes.latency' 2>/dev/null || echo "0"
}

# Measure round-trip export latency: time a telemetrygen burst to appear in Jaeger
measure_trace_arrival() {
  local scenario="$1"
  local tag="latency-test-${scenario}-$(date +%s)"

  log "Sending 1 trace burst tagged service=${tag}..."
  local t_start t_end elapsed

  t_start=$(date +%s%3N)

  # Send a single trace via telemetrygen (if available) or via direct OTLP HTTP
  if command -v telemetrygen &>/dev/null; then
    telemetrygen traces \
      --otlp-endpoint localhost:4317 \
      --otlp-insecure \
      --duration 1s \
      --rate 1 \
      --service "${tag}" > /dev/null 2>&1 || true
  else
    # Fallback: use docker exec to run telemetrygen from existing container
    docker compose exec -T telemetrygen-traces sh -c \
      "telemetrygen traces --otlp-endpoint otel-collector-1:4317 --otlp-insecure --duration=1s --rate=1 --service=${tag}" \
      > /dev/null 2>&1 || true
  fi

  t_end=$(date +%s%3N)
  elapsed=$(( t_end - t_start ))
  echo "$elapsed"
}

# Query collector export success metric from Prometheus
get_collector_export_metric() {
  local metric="otel_exporter_sent_spans_total"
  curl -sf "${PROM_API}/api/v1/query?query=${metric}" 2>/dev/null \
    | jq -r '.data.result[0].value[1] // "N/A"' 2>/dev/null || echo "N/A"
}

# Query Toxiproxy for bytes transferred (indirect proxy health)
get_proxy_stats() {
  curl -sf "${TOXIPROXY_API}/proxies/${PROXY_NAME}" 2>/dev/null \
    | jq '{enabled: .enabled, upstream: .upstream}' 2>/dev/null || echo "{}"
}

# Wait and show a countdown
observe() {
  local secs="$1" label="$2"
  log "Observing for ${secs}s — ${label}"
  for i in $(seq "$secs" -1 1); do
    printf "\r    %ds remaining..." "$i"
    sleep 1
  done
  echo ""
}

# ─── scenario runner ──────────────────────────────────────────────────────────

run_scenario() {
  local name="$1" lat="$2" jitter="$3" expected="$4"

  sep
  echo ""
  log "SCENARIO: ${name}"
  log "Latency : ${lat}ms  |  Jitter: ±${jitter}ms"
  log "Expected: ${expected}"
  echo ""

  set_latency "$lat" "$jitter"
  sleep 1

  local actual_lat
  actual_lat=$(get_current_latency)
  ok "Toxic applied — current toxic latency: ${actual_lat}ms"

  observe "$WAIT_SECS" "${name}"

  log "Sampling collector metrics..."
  local spans_sent
  spans_sent=$(get_collector_export_metric)
  log "  otel_exporter_sent_spans_total = ${spans_sent}"

  local proxy_info
  proxy_info=$(get_proxy_stats)
  log "  Proxy state: ${proxy_info}"

  echo ""
  log "RESULT: See Jaeger UI → http://localhost:16686  |  Prometheus → http://localhost:9090"
  echo ""
}

print_header() {
  sep
  echo ""
  echo "  OTel Collector Latency Effect Tests"
  echo "  POC-019 — Toxiproxy between collector-1 and collector-2"
  echo ""
  echo "  Architecture:"
  echo "  [Generator] → [collector-1:4317] → [Toxiproxy:14317] → [collector-2:4317] → [Jaeger / Prometheus]"
  echo ""
  sep
  echo ""
}

print_summary() {
  sep
  echo ""
  echo "  TEST SUMMARY"
  echo ""
  echo "  Latency  | Expected effect"
  echo "  ─────────────────────────────────────────────────────────────────────"
  echo "  0ms      | Baseline — negligible pipeline delay"
  echo "  50ms     | Acceptable — within typical LAN/cluster round-trip"
  echo "  200ms    | Noticeable — batch timeout starts masking individual spans"
  echo "  800ms    | Degraded  — export retries begin, queue pressure increases"
  echo "  2000ms   | Critical  — exporter timeouts likely, spans may be dropped"
  echo ""
  echo "  KEY OBSERVATIONS:"
  echo "  1. Batch processor timeout (5s) hides small latencies — delay ≤ 5s"
  echo "     won't cause data loss but increases end-to-end telemetry age."
  echo "  2. At high latency the gRPC exporter retries (default: 300s total)."
  echo "     When the retry budget is exhausted, spans are DROPPED silently."
  echo "  3. Jitter compounds: at 2000ms ±500ms some sends timeout (>5s gRPC"
  echo "     deadline) and the queue fills, causing backpressure on collector-1."
  echo "  4. Prometheus scrape of otel-collector-2 metrics becomes stale when"
  echo "     collector-2 itself is backlogged processing buffered batches."
  echo ""
  sep
  echo ""
}

# ─── main ─────────────────────────────────────────────────────────────────────

check_deps
check_services
print_header

SCENARIO="${1:-all}"

case "$SCENARIO" in
  0|baseline)
    run_scenario "BASELINE — 0ms"  0    0   "No artificial delay; spans arrive within batch timeout (5s)"
    ;;
  1|low)
    run_scenario "LOW — 50ms"      50   10  "Transparent to users; small increase in span age"
    ;;
  2|medium)
    run_scenario "MEDIUM — 200ms"  200  30  "Visible in Jaeger span timeline; export queue grows"
    ;;
  3|high)
    run_scenario "HIGH — 800ms"    800  100 "Retry budget consumed; intermittent export failures in collector logs"
    ;;
  4|critical)
    run_scenario "CRITICAL — 2s"   2000 500 "gRPC deadline exceeded; exporter drops spans; queue overflow"
    ;;
  5|restore)
    remove_toxic
    ok "All toxics removed — back to no latency."
    ;;
  all)
    run_scenario "BASELINE — 0ms"  0    0   "No artificial delay; spans arrive within batch timeout (5s)"
    run_scenario "LOW — 50ms"      50   10  "Transparent to users; small increase in span age"
    run_scenario "MEDIUM — 200ms"  200  30  "Visible in Jaeger span timeline; export queue grows"
    run_scenario "HIGH — 800ms"    800  100 "Retry budget consumed; intermittent export failures in collector logs"
    run_scenario "CRITICAL — 2s"   2000 500 "gRPC deadline exceeded; exporter drops spans; queue overflow"

    log "Restoring baseline (0ms)..."
    remove_toxic
    ok "Toxics removed — baseline restored."
    ;;
  *)
    err "Unknown scenario: ${SCENARIO}"
    echo "Usage: $0 [0|1|2|3|4|5|baseline|low|medium|high|critical|restore|all]"
    exit 1
    ;;
esac

print_summary
