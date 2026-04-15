#!/usr/bin/env bash
# Smoke-tests the POC:
#   1. Gateway LB IP is assigned
#   2. canary.local serves nginx (v1 — red)
#   3. bluegreen.local serves nginx (blue)
#   4. Prometheus reachable and kube-state-metrics has our pods
#   5. Argo Rollouts controller healthy
set -euo pipefail
CLUSTER=${CLUSTER:-rollouts}
k() { kubectl --context "kind-${CLUSTER}" "$@"; }

fail=0
ok()  { printf "  \033[1;32mOK\033[0m   %s\n" "$*"; }
bad() { printf "  \033[1;31mFAIL\033[0m %s\n" "$*"; fail=1; }

echo "==> 1. Envoy Gateway LB IP"
GW_IP=$(k -n envoy-gateway-system get svc -l gateway.envoyproxy.io/owning-gateway-name=eg \
  -o jsonpath='{.items[0].status.loadBalancer.ingress[0].ip}')
if [[ -n "$GW_IP" ]]; then ok "gateway IP = $GW_IP"; else bad "no LB IP"; exit 1; fi

echo "==> 2. canary.local responds with a known revision"
body=$(curl -s --max-time 5 -H 'Host: canary.local' "http://${GW_IP}/")
if   echo "$body" | grep -q "v1 — red";   then ok "canary serves v1 (red)"
elif echo "$body" | grep -q "v2 — green"; then ok "canary serves v2 (green)"
else bad "canary returned unexpected body: $body"; fi

echo "==> 3. bluegreen.local responds with blue or green"
body=$(curl -s --max-time 5 -H 'Host: bluegreen.local' "http://${GW_IP}/")
if   echo "$body" | grep -q "BLUE";  then ok "bluegreen active serves BLUE"
elif echo "$body" | grep -q "GREEN"; then ok "bluegreen active serves GREEN"
else bad "bluegreen returned unexpected body: $body"; fi

echo "==> 4. Prometheus reachable + kube-state-metrics sees app-canary pods"
q='count(kube_pod_info{namespace="app-canary"})'
res=$(k -n monitoring exec svc/kps-prometheus -c prometheus -- \
  wget -qO- "http://localhost:9090/api/v1/query?query=${q// /}" || true)
if echo "$res" | grep -q '"status":"success"'; then
  ok "prometheus query succeeded: $(echo "$res" | sed 's/.*"value":\[[0-9.]*,"\([0-9]*\)".*/\1 pods/')"
else
  bad "prometheus query failed"
fi

echo "==> 5. Argo Rollouts controller healthy"
ready=$(k -n argo-rollouts get deploy argo-rollouts -o jsonpath='{.status.readyReplicas}')
desired=$(k -n argo-rollouts get deploy argo-rollouts -o jsonpath='{.spec.replicas}')
if [[ -n "$ready" && "$ready" == "$desired" ]]; then
  ok "argo-rollouts deployment ready"
else
  bad "argo-rollouts deployment not ready"
fi

echo
[[ $fail -eq 0 ]] && { echo "ALL GOOD"; exit 0; } || { echo "SOMETHING BROKE"; exit 1; }
