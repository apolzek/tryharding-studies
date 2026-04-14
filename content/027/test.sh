#!/usr/bin/env bash
# Exercises the mesh. Calls helloworld from the sleep pod on both clusters and
# asserts that responses come back from both v1 and v2 — i.e. Istio is load
# balancing across cluster boundaries through the east-west gateway.
set -euo pipefail

EAST_CTX=kind-east
WEST_CTX=kind-west

log() { printf '\033[1;36m==> %s\033[0m\n' "$*"; }

call() {
  local ctx=$1
  kubectl --context "${ctx}" -n sample exec deploy/sleep -c sleep -- \
    curl -sS helloworld.sample:5000/hello
}

run_iter() {
  local ctx=$1
  local -A seen=()
  for _ in $(seq 1 20); do
    out=$(call "${ctx}" || true)
    if [[ "${out}" == *"version: v1"* ]]; then seen[v1]=1; fi
    if [[ "${out}" == *"version: v2"* ]]; then seen[v2]=1; fi
    echo "  ${out}"
  done
  if [[ -n "${seen[v1]:-}" && -n "${seen[v2]:-}" ]]; then
    log "[${ctx}] SAW v1 AND v2 — cross-cluster traffic confirmed"
  else
    echo "[${ctx}] FAILED: only saw: ${!seen[*]}"
    return 1
  fi
}

log "Calling helloworld from sleep on east cluster"
run_iter "${EAST_CTX}"
log "Calling helloworld from sleep on west cluster"
run_iter "${WEST_CTX}"

log "Istio remote secrets seen by each istiod"
kubectl --context "${EAST_CTX}" -n istio-system get secrets -l istio/multiCluster=true
kubectl --context "${WEST_CTX}" -n istio-system get secrets -l istio/multiCluster=true

log "East-west gateway LB IPs"
kubectl --context "${EAST_CTX}" -n istio-system get svc istio-eastwestgateway
kubectl --context "${WEST_CTX}" -n istio-system get svc istio-eastwestgateway

log "Grafana is at http://localhost:30300 (admin/admin)"
