#!/usr/bin/env bash
# Full POC installer: kind cluster + MetalLB + Gateway API CRDs + Envoy Gateway
# + kube-prometheus-stack + Argo CD + Argo Rollouts + two nginx rollouts
# (canary and blue/green) exposed via HTTPRoutes.
#
# Idempotent: re-runnable; helm upgrade --install is used throughout.
set -euo pipefail

CLUSTER=${CLUSTER:-rollouts}
HERE="$(cd "$(dirname "$0")" && pwd)"
cd "$HERE"

# Pinned versions -----------------------------------------------------------
NODE_IMAGE=kindest/node:v1.34.0
METALLB_CHART_VERSION=0.15.3
GATEWAY_API_VERSION=v1.3.0
ENVOY_GATEWAY_VERSION=v1.5.1
KPS_VERSION=83.4.2
ARGOCD_CHART_VERSION=9.5.0
ROLLOUTS_CHART_VERSION=2.40.9
# ---------------------------------------------------------------------------

log() { printf "\n\033[1;36m==>\033[0m %s\n" "$*"; }

k() { kubectl --context "kind-${CLUSTER}" "$@"; }

create_cluster() {
  if kind get clusters | grep -qx "$CLUSTER"; then
    log "kind cluster '${CLUSTER}' already exists — skipping create"
  else
    log "creating kind cluster '${CLUSTER}' (${NODE_IMAGE})"
    kind create cluster --name "$CLUSTER" --image "$NODE_IMAGE" --config kind.yaml --wait 120s
  fi
}

install_metallb() {
  log "installing MetalLB ${METALLB_CHART_VERSION}"
  helm repo add metallb https://metallb.github.io/metallb >/dev/null 2>&1 || true
  helm repo update metallb >/dev/null
  helm --kube-context "kind-${CLUSTER}" upgrade --install metallb metallb/metallb \
    --version "$METALLB_CHART_VERSION" \
    --namespace metallb-system --create-namespace \
    --wait --timeout 5m
  log "applying MetalLB IPAddressPool (172.18.255.200-220)"
  until k apply -f metallb-pool.yaml 2>/dev/null; do sleep 3; done
}

install_envoy_gateway() {
  # Envoy Gateway's Helm chart bundles the Gateway API v1.3 standard-channel
  # CRDs already — installing them separately first clashes on field
  # ownership (kubectl-client-side-apply vs helm server-side-apply), so we
  # let the chart own them.
  log "installing Envoy Gateway ${ENVOY_GATEWAY_VERSION} (bundles Gateway API ${GATEWAY_API_VERSION} CRDs)"
  helm --kube-context "kind-${CLUSTER}" upgrade --install eg \
    oci://docker.io/envoyproxy/gateway-helm \
    --version "${ENVOY_GATEWAY_VERSION}" \
    --namespace envoy-gateway-system --create-namespace \
    --wait --timeout 5m
  log "creating GatewayClass + Gateway"
  k apply -f envoy-gateway.yaml
  log "waiting for Gateway to be programmed"
  k -n envoy-gateway-system wait gateway/eg --for=condition=Programmed --timeout=3m || true
}

install_kps() {
  log "installing kube-prometheus-stack ${KPS_VERSION}"
  helm repo add prometheus-community https://prometheus-community.github.io/helm-charts >/dev/null 2>&1 || true
  helm repo update prometheus-community >/dev/null
  k get ns monitoring >/dev/null 2>&1 || k create ns monitoring
  helm --kube-context "kind-${CLUSTER}" upgrade --install kps prometheus-community/kube-prometheus-stack \
    --version "$KPS_VERSION" \
    --namespace monitoring \
    -f values-kube-prometheus-stack.yaml \
    --wait --timeout 10m
}

install_argocd() {
  log "installing Argo CD ${ARGOCD_CHART_VERSION}"
  helm repo add argo https://argoproj.github.io/argo-helm >/dev/null 2>&1 || true
  helm repo update argo >/dev/null
  k get ns argocd >/dev/null 2>&1 || k create ns argocd
  helm --kube-context "kind-${CLUSTER}" upgrade --install argocd argo/argo-cd \
    --version "$ARGOCD_CHART_VERSION" \
    --namespace argocd \
    -f values-argocd.yaml \
    --wait --timeout 10m
}

install_rollouts() {
  log "installing Argo Rollouts ${ROLLOUTS_CHART_VERSION}"
  k get ns argo-rollouts >/dev/null 2>&1 || k create ns argo-rollouts
  helm --kube-context "kind-${CLUSTER}" upgrade --install argo-rollouts argo/argo-rollouts \
    --version "$ROLLOUTS_CHART_VERSION" \
    --namespace argo-rollouts \
    --set dashboard.enabled=true \
    --wait --timeout 10m

  # kubectl argo rollouts plugin (optional — everything in test.sh and the
  # README works with plain kubectl, but the plugin gives a nicer CLI)
  local bin="${HERE}/bin"; mkdir -p "$bin"
  if [[ ! -x "${bin}/kubectl-argo-rollouts" ]]; then
    log "downloading kubectl-argo-rollouts plugin"
    local v="v1.8.3"
    curl -sL -o "${bin}/kubectl-argo-rollouts" \
      "https://github.com/argoproj/argo-rollouts/releases/download/${v}/kubectl-argo-rollouts-linux-amd64"
    chmod +x "${bin}/kubectl-argo-rollouts"
  fi
  echo "    add to PATH:   export PATH=\"${bin}:\$PATH\""
}

deploy_apps() {
  log "deploying nginx-canary Rollout"
  k apply -f app-canary.yaml
  log "deploying nginx-bluegreen Rollout"
  k apply -f app-bluegreen.yaml
  log "waiting for canary stable rollout"
  k -n app-canary wait rollout/nginx-canary --for=condition=Available=true --timeout=3m || true
  k -n app-bluegreen wait rollout/nginx-bg    --for=condition=Available=true --timeout=3m || true
}

print_access() {
  local gw_ip
  gw_ip=$(k -n envoy-gateway-system get svc -l gateway.envoyproxy.io/owning-gateway-name=eg -o jsonpath='{.items[0].status.loadBalancer.ingress[0].ip}' 2>/dev/null || true)
  cat <<EOF

========================================================================
 READY
========================================================================
 Envoy Gateway LB IP : ${gw_ip:-<pending>}

 HTTP access to the apps (Host header required):
   curl -H 'Host: canary.local'    http://${gw_ip:-<gw-ip>}/
   curl -H 'Host: bluegreen.local' http://${gw_ip:-<gw-ip>}/

 UIs via kind port mapping on localhost:
   Grafana     http://localhost:31300   (admin / admin)
   Prometheus  http://localhost:31090
   Argo CD     http://localhost:31880
                 user: admin
                 pass: \$(kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath='{.data.password}' | base64 -d)

 Rollouts CLI / dashboard:
   kubectl argo rollouts get rollout nginx-canary -n app-canary -w
   kubectl argo rollouts dashboard -n argo-rollouts    # http://localhost:3100
========================================================================
EOF
}

main() {
  create_cluster
  install_metallb
  install_envoy_gateway
  install_kps
  install_argocd
  install_rollouts
  deploy_apps
  print_access
}

main "$@"
