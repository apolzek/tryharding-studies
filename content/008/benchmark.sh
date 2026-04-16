#!/usr/bin/env bash
# Benchmarks Prometheus under two scrape configurations (baseline vs. mitigated)
# and prints a markdown table comparing head-series count, resident memory,
# ingest rate and query latency.
#
# Requirements: docker, docker compose, curl, jq, bc, python3

set -euo pipefail

cd "$(dirname "$0")"

WARMUP_SECONDS="${WARMUP_SECONDS:-90}"
QUERY_RUNS="${QUERY_RUNS:-5}"
PROM_URL="http://localhost:9090"

# URL-encode a PromQL expression using python3 (avoids manual escaping).
urlencode() {
  python3 -c 'import sys,urllib.parse;print(urllib.parse.quote(sys.argv[1],safe=""))' "$1"
}

# Scalar instant-query against Prometheus. Returns raw string value.
query_scalar() {
  local expr encoded
  expr="$1"
  encoded=$(urlencode "$expr")
  curl -sf "${PROM_URL}/api/v1/query?query=${encoded}" \
    | jq -r '.data.result[0].value[1] // "NaN"'
}

# Average time_total (seconds) of N identical queries, formatted as 0.NNN.
query_latency() {
  local expr encoded total=0 t avg
  expr="$1"
  encoded=$(urlencode "$expr")
  for _ in $(seq 1 "$QUERY_RUNS"); do
    t=$(curl -sf -o /dev/null -w "%{time_total}" \
         "${PROM_URL}/api/v1/query?query=${encoded}")
    total=$(echo "$total + $t" | bc -l)
  done
  avg=$(echo "scale=3; $total / $QUERY_RUNS" | bc -l)
  printf "%.3f" "$avg"
}

wait_until_ready() {
  local tries=0
  until curl -sf "${PROM_URL}/-/ready" >/dev/null 2>&1; do
    tries=$((tries + 1))
    if [ $tries -gt 60 ]; then
      echo "Prometheus never became ready" >&2
      return 1
    fi
    sleep 1
  done
}

run_scenario() {
  local name="$1"
  local config="$2"

  echo "[$(date +%H:%M:%S)] === $name ($config) ===" >&2

  PROMETHEUS_CONFIG="$config" docker compose up -d --build >/dev/null
  wait_until_ready

  echo "[$(date +%H:%M:%S)] warming up for ${WARMUP_SECONDS}s..." >&2
  sleep "$WARMUP_SECONDS"

  local head_series rss_bytes rss_mb ingest_raw ingest
  head_series=$(query_scalar 'prometheus_tsdb_head_series')
  rss_bytes=$(query_scalar 'process_resident_memory_bytes{job="prometheus"}')
  rss_mb=$(echo "scale=0; $rss_bytes / 1048576" | bc)
  ingest_raw=$(query_scalar 'sum(rate(prometheus_tsdb_head_samples_appended_total[1m]))')
  ingest=$(printf "%.0f" "$ingest_raw")

  local q_all q_regex q_topk
  q_all=$(query_latency 'count({__name__=~".+"})')
  q_regex=$(query_latency 'count({__name__=~".+", job="genmetrics"})')
  q_topk=$(query_latency 'topk(10, count by (__name__)({__name__=~".+"}))')

  docker compose down -v >/dev/null 2>&1

  # emit one tab-separated row; aggregator below turns this into a table
  printf "%s\t%.0f\t%s\t%s\t%s\t%s\t%s\n" \
    "$name" "$head_series" "${rss_mb} MB" \
    "${ingest}/s" "${q_all}s" "${q_regex}s" "${q_topk}s"
}

main() {
  local tmp
  tmp=$(mktemp)
  # shellcheck disable=SC2064
  trap "rm -f '$tmp'" EXIT

  {
    run_scenario "Baseline"  "prometheus.yml"
    run_scenario "Mitigated" "prometheus-mitigated.yml"
  } | tee "$tmp" >/dev/null

  {
    echo "| Scenario  | Head series | RSS | Samples ingested | count(all) | count(job) | topk(10) |"
    echo "|-----------|------------:|----:|-----------------:|-----------:|-----------:|---------:|"
    while IFS=$'\t' read -r name series rss ingest q1 q2 q3; do
      printf "| %-9s | %11s | %3s | %16s | %10s | %10s | %8s |\n" \
        "$name" "$series" "$rss" "$ingest" "$q1" "$q2" "$q3"
    done < "$tmp"
  } | tee BENCHMARK.md
}

main "$@"
