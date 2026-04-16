#!/bin/bash
# test-latency.sh — Does injected latency between two OTel collectors produce queue buildup?
#
# Runs every command via `docker compose exec` on a helper `client` container (alpine + curl + jq),
# so no tools are required on the host besides docker and docker compose.
#
# For each scenario the script:
#   1. clears any existing toxic
#   2. waits for the sending queue to drain
#   3. applies the latency toxic
#   4. samples otelcol_exporter_queue_size on collector-1 every few seconds
#   5. records the peak queue size and any send failures
#   6. prints a comparison table at the end
#
# Usage:
#   ./test-latency.sh            # run all scenarios
#   ./test-latency.sh medium     # run a single scenario
#   ./test-latency.sh restore    # remove all toxics

set -euo pipefail

COMPOSE=(docker compose)
CLIENT=("${COMPOSE[@]}" exec -T client)
TOXIPROXY_URL="http://toxiproxy:8474"
COLLECTOR1_METRICS="http://otel-collector-1:8888/metrics"
PROXY_NAME="otel-collector"
TOXIC_NAME="latency"

OBSERVE_SECS=45        # total time each scenario runs
SAMPLE_INTERVAL=3      # seconds between queue_size samples
DRAIN_SECS=15          # wait between scenarios so queue goes back to baseline

# ─── helpers ──────────────────────────────────────────────────────────────────

log()  { echo -e "\033[1;34m[INFO]\033[0m  $*"; }
ok()   { echo -e "\033[1;32m[ OK ]\033[0m  $*"; }
warn() { echo -e "\033[1;33m[WARN]\033[0m  $*"; }
err()  { echo -e "\033[1;31m[ERR ]\033[0m  $*"; }
sep()  { echo -e "\033[1;37m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\033[0m"; }

# Run a command inside the helper client container.
client_exec() {
  "${CLIENT[@]}" sh -c "$*"
}

check_stack() {
  log "Verifying the compose stack is up..."
  if ! "${COMPOSE[@]}" ps --status=running --services | grep -q '^client$'; then
    err "The 'client' service is not running. Bring the stack up first:"
    err "    docker compose up -d"
    exit 1
  fi
  client_exec "curl -sf ${TOXIPROXY_URL}/proxies > /dev/null" \
    && ok "Toxiproxy reachable from client container" \
    || { err "Toxiproxy not reachable from client. Is the stack healthy?"; exit 1; }
  client_exec "curl -sf ${COLLECTOR1_METRICS} > /dev/null" \
    && ok "Collector-1 internal metrics reachable" \
    || { err "Collector-1 :8888/metrics not reachable"; exit 1; }
  echo ""
}

# Extract a single numeric metric value (first matching line) from /metrics.
scrape_metric() {
  local metric="$1"
  client_exec "curl -sf ${COLLECTOR1_METRICS} \
    | grep -E '^${metric}[ {]' \
    | awk '{ sum += \$NF } END { if (NR==0) print 0; else print sum }'"
}

remove_toxic() {
  client_exec "curl -sf -X DELETE ${TOXIPROXY_URL}/proxies/${PROXY_NAME}/toxics/${TOXIC_NAME} > /dev/null 2>&1 || true"
}

apply_toxic() {
  local lat="$1" jitter="$2"
  remove_toxic
  [[ "$lat" -eq 0 ]] && return 0
  client_exec "curl -sf -X POST ${TOXIPROXY_URL}/proxies/${PROXY_NAME}/toxics \
    -H 'Content-Type: application/json' \
    -d '{\"name\":\"${TOXIC_NAME}\",\"type\":\"latency\",\"stream\":\"upstream\",\"toxicity\":1.0,\"attributes\":{\"latency\":${lat},\"jitter\":${jitter}}}' \
    > /dev/null"
}

# Sample queue_size every SAMPLE_INTERVAL seconds for OBSERVE_SECS and print peak.
observe_queue() {
  local secs="$1"
  local peak=0 cur sample
  local end=$(( $(date +%s) + secs ))
  while [[ $(date +%s) -lt $end ]]; do
    cur=$(scrape_metric "otelcol_exporter_queue_size" | tr -d '[:space:]')
    cur=${cur%.*}
    [[ -z "$cur" ]] && cur=0
    (( cur > peak )) && peak=$cur
    printf "\r    queue_size = %4d   (peak so far: %4d)   %ds left " \
      "$cur" "$peak" "$(( end - $(date +%s) ))"
    sleep "$SAMPLE_INTERVAL"
  done
  echo ""
  echo "$peak"
}

# ─── scenario runner ──────────────────────────────────────────────────────────

declare -a RESULTS

run_scenario() {
  local name="$1" lat="$2" jitter="$3"

  sep
  log "SCENARIO: ${name}   latency=${lat}ms   jitter=±${jitter}ms"
  echo ""

  log "Clearing toxic and draining queue for ${DRAIN_SECS}s..."
  remove_toxic
  sleep "$DRAIN_SECS"

  local q_before failed_before
  q_before=$(scrape_metric "otelcol_exporter_queue_size" | tr -d '[:space:]')
  failed_before=$(scrape_metric "otelcol_exporter_send_failed_spans_total" | tr -d '[:space:]')
  failed_before=${failed_before%.*}
  [[ -z "$failed_before" ]] && failed_before=0
  log "Before: queue_size=${q_before}   failed_spans_total=${failed_before}"

  apply_toxic "$lat" "$jitter"
  ok "Toxic applied (latency=${lat}ms, jitter=${jitter}ms)"

  local peak
  peak=$(observe_queue "$OBSERVE_SECS")

  local failed_after
  failed_after=$(scrape_metric "otelcol_exporter_send_failed_spans_total" | tr -d '[:space:]')
  failed_after=${failed_after%.*}
  [[ -z "$failed_after" ]] && failed_after=0
  local failed_delta=$(( failed_after - failed_before ))

  ok "Peak queue_size during scenario: ${peak}"
  ok "send_failed_spans during scenario: ${failed_delta}"
  echo ""

  RESULTS+=("$(printf '%-10s %-10s %-10s %-18s %-10s' \
    "$name" "${lat}ms" "±${jitter}ms" "$peak" "$failed_delta")")
}

print_summary() {
  sep
  echo ""
  echo "  QUEUE BUILDUP BY SCENARIO"
  echo ""
  printf "  %-10s %-10s %-10s %-18s %-10s\n" \
    "scenario" "latency" "jitter" "peak queue_size" "failed_spans"
  echo "  ─────────────────────────────────────────────────────────────────────"
  for row in "${RESULTS[@]}"; do
    printf "  %s\n" "$row"
  done
  echo ""
  echo "  Answer the question: 'does added latency build up the exporter queue?'"
  echo "  • peak queue_size stays near 0  → latency absorbed by batch + gRPC"
  echo "  • peak queue_size grows         → latency slows the exporter; pressure builds"
  echo "  • failed_spans > 0              → retry budget exhausted, items dropped"
  echo ""
  sep
}

# ─── main ─────────────────────────────────────────────────────────────────────

check_stack

SCENARIO="${1:-all}"

case "$SCENARIO" in
  baseline) run_scenario "baseline" 0   0   ;;
  low)      run_scenario "low"      50  10  ;;
  medium)   run_scenario "medium"   200 30  ;;
  high)     run_scenario "high"     800 100 ;;
  critical) run_scenario "critical" 2000 500 ;;
  restore)  remove_toxic; ok "All toxics removed."; exit 0 ;;
  all)
    run_scenario "baseline" 0    0
    run_scenario "low"      50   10
    run_scenario "medium"   200  30
    run_scenario "high"     800  100
    run_scenario "critical" 2000 500
    remove_toxic
    ok "All toxics removed — stack back to baseline."
    ;;
  *)
    err "Unknown scenario: ${SCENARIO}"
    echo "Usage: $0 [baseline|low|medium|high|critical|restore|all]"
    exit 1
    ;;
esac

print_summary
