#!/usr/bin/env bash
# Bootstraps two kind clusters joined into a single Istio multi-primary /
# multi-network mesh, plus MetalLB, kube-prometheus-stack and Grafana
# dashboards for Istio observability.
set -euo pipefail

ISTIO_VERSION="${ISTIO_VERSION:-1.29.2}"
METALLB_VERSION="${METALLB_VERSION:-v0.14.8}"
KPS_VERSION="${KPS_VERSION:-83.4.2}"

EAST=east
WEST=west
EAST_CTX="kind-${EAST}"
WEST_CTX="kind-${WEST}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CERTS_DIR="${SCRIPT_DIR}/certs"

log() { printf '\033[1;34m==> %s\033[0m\n' "$*"; }

############################################
# 1. kind clusters
############################################
create_clusters() {
  log "Creating kind cluster ${EAST}"
  kind create cluster --config "${SCRIPT_DIR}/kind-east.yaml"
  log "Creating kind cluster ${WEST}"
  kind create cluster --config "${SCRIPT_DIR}/kind-west.yaml"

  log "Waiting for cluster nodes to be Ready"
  kubectl --context "${EAST_CTX}" wait --for=condition=Ready nodes --all --timeout=180s
  kubectl --context "${WEST_CTX}" wait --for=condition=Ready nodes --all --timeout=180s
}

############################################
# 2. MetalLB (required so Istio east-west gateway gets an external IP)
############################################
install_metallb() {
  for ctx in "${EAST_CTX}" "${WEST_CTX}"; do
    log "Installing MetalLB on ${ctx}"
    kubectl --context "${ctx}" apply -f \
      "https://raw.githubusercontent.com/metallb/metallb/${METALLB_VERSION}/config/manifests/metallb-native.yaml"
  done

  for ctx in "${EAST_CTX}" "${WEST_CTX}"; do
    kubectl --context "${ctx}" -n metallb-system rollout status deploy/controller --timeout=180s
    kubectl --context "${ctx}" -n metallb-system wait --for=condition=Ready pods -l app=metallb --timeout=180s
  done

  kubectl --context "${EAST_CTX}" apply -f "${SCRIPT_DIR}/metallb-east.yaml"
  kubectl --context "${WEST_CTX}" apply -f "${SCRIPT_DIR}/metallb-west.yaml"
}

############################################
# 3. Shared root CA for both clusters (cacerts secret)
############################################
generate_ca() {
  if [[ -f "${CERTS_DIR}/cert-chain.pem" ]]; then
    log "Reusing existing CA in ${CERTS_DIR}"
    return
  fi
  log "Generating shared root CA under ${CERTS_DIR}"
  mkdir -p "${CERTS_DIR}"
  cd "${CERTS_DIR}"

  openssl genrsa -out root-key.pem 4096
  openssl req -x509 -new -nodes -key root-key.pem -days 3650 -sha256 \
    -out root-cert.pem -subj "/O=Istio/CN=Root CA"

  openssl genrsa -out ca-key.pem 4096
  openssl req -new -key ca-key.pem -out intermediate.csr \
    -subj "/O=Istio/CN=Intermediate CA"

  cat > intermediate.ext <<EOF
basicConstraints = critical, CA:TRUE, pathlen:0
keyUsage = critical, digitalSignature, keyCertSign, cRLSign
subjectKeyIdentifier = hash
EOF
  openssl x509 -req -in intermediate.csr -CA root-cert.pem -CAkey root-key.pem \
    -CAcreateserial -out ca-cert.pem -days 3650 -sha256 \
    -extfile intermediate.ext

  cat ca-cert.pem root-cert.pem > cert-chain.pem
  cd - >/dev/null
}

install_cacerts() {
  for ctx in "${EAST_CTX}" "${WEST_CTX}"; do
    log "Installing cacerts on ${ctx}"
    kubectl --context "${ctx}" create namespace istio-system --dry-run=client -o yaml | \
      kubectl --context "${ctx}" apply -f -
    kubectl --context "${ctx}" -n istio-system delete secret cacerts --ignore-not-found
    kubectl --context "${ctx}" -n istio-system create secret generic cacerts \
      --from-file="${CERTS_DIR}/ca-cert.pem" \
      --from-file="${CERTS_DIR}/ca-key.pem" \
      --from-file="${CERTS_DIR}/root-cert.pem" \
      --from-file="${CERTS_DIR}/cert-chain.pem"
  done
}

############################################
# 4. Istio multi-primary / multi-network via Helm
############################################
label_network() {
  local ctx=$1 net=$2
  kubectl --context "${ctx}" label namespace istio-system \
    topology.istio.io/network="${net}" --overwrite
}

helm_istio() {
  local ctx=$1 values=$2 ewvalues=$3 net=$4
  helm repo add istio https://istio-release.storage.googleapis.com/charts >/dev/null 2>&1 || true
  helm repo update >/dev/null

  label_network "${ctx}" "${net}"

  log "[${ctx}] helm install istio-base"
  helm --kube-context "${ctx}" upgrade --install istio-base istio/base \
    -n istio-system --version "${ISTIO_VERSION}" --set defaultRevision=default

  log "[${ctx}] helm install istiod"
  helm --kube-context "${ctx}" upgrade --install istiod istio/istiod \
    -n istio-system --version "${ISTIO_VERSION}" -f "${values}"

  kubectl --context "${ctx}" -n istio-system rollout status deploy/istiod --timeout=240s

  log "[${ctx}] helm install istio-eastwestgateway"
  helm --kube-context "${ctx}" upgrade --install istio-eastwestgateway istio/gateway \
    -n istio-system --version "${ISTIO_VERSION}" -f "${ewvalues}"

  kubectl --context "${ctx}" -n istio-system rollout status deploy/istio-eastwestgateway --timeout=240s
}

install_istio() {
  helm_istio "${EAST_CTX}" "${SCRIPT_DIR}/values-istiod-east.yaml" \
    "${SCRIPT_DIR}/values-eastwest-east.yaml" "network-east"
  helm_istio "${WEST_CTX}" "${SCRIPT_DIR}/values-istiod-west.yaml" \
    "${SCRIPT_DIR}/values-eastwest-west.yaml" "network-west"

  log "Applying cross-network gateway on both clusters"
  kubectl --context "${EAST_CTX}" apply -n istio-system -f "${SCRIPT_DIR}/expose-services.yaml"
  kubectl --context "${WEST_CTX}" apply -n istio-system -f "${SCRIPT_DIR}/expose-services.yaml"
}

############################################
# 5. Endpoint discovery: each cluster gets the other cluster's kubeconfig
############################################
install_istioctl() {
  if command -v "${SCRIPT_DIR}/bin/istioctl" >/dev/null 2>&1; then
    return
  fi
  log "Downloading istioctl ${ISTIO_VERSION}"
  mkdir -p "${SCRIPT_DIR}/bin"
  (
    cd /tmp
    curl -sSL "https://github.com/istio/istio/releases/download/${ISTIO_VERSION}/istio-${ISTIO_VERSION}-linux-amd64.tar.gz" \
      | tar -xz
    mv "istio-${ISTIO_VERSION}/bin/istioctl" "${SCRIPT_DIR}/bin/istioctl"
    rm -rf "istio-${ISTIO_VERSION}"
  )
}

link_clusters() {
  install_istioctl
  local istioctl="${SCRIPT_DIR}/bin/istioctl"

  # Kind places each control plane API on a random host port. For remote secrets
  # Istio needs a *server address reachable from the other cluster*. We rewrite
  # the kubeconfig to use the internal docker-network IP of the control-plane.
  local east_ip west_ip
  east_ip=$(docker inspect east-control-plane -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}')
  west_ip=$(docker inspect west-control-plane -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}')

  log "Creating remote secret for east -> apply on west (server https://${east_ip}:6443)"
  "${istioctl}" --context "${EAST_CTX}" create-remote-secret \
    --name=east \
    --server="https://${east_ip}:6443" \
    | kubectl --context "${WEST_CTX}" apply -f -

  log "Creating remote secret for west -> apply on east (server https://${west_ip}:6443)"
  "${istioctl}" --context "${WEST_CTX}" create-remote-secret \
    --name=west \
    --server="https://${west_ip}:6443" \
    | kubectl --context "${EAST_CTX}" apply -f -
}

############################################
# 6. Observability (kube-prometheus-stack + Istio dashboards) on east
############################################
install_observability() {
  helm repo add prometheus-community https://prometheus-community.github.io/helm-charts >/dev/null 2>&1 || true
  helm repo update >/dev/null

  log "[${EAST_CTX}] Installing kube-prometheus-stack"
  kubectl --context "${EAST_CTX}" create ns monitoring --dry-run=client -o yaml | \
    kubectl --context "${EAST_CTX}" apply -f -
  helm --kube-context "${EAST_CTX}" upgrade --install kps prometheus-community/kube-prometheus-stack \
    -n monitoring --version "${KPS_VERSION}" \
    -f "${SCRIPT_DIR}/values-kube-prometheus-stack.yaml"
  kubectl --context "${EAST_CTX}" -n monitoring rollout status deploy/kps-grafana --timeout=300s
}

############################################
# 7. Sample application split across the two clusters
############################################
deploy_sample() {
  for ctx in "${EAST_CTX}" "${WEST_CTX}"; do
    log "[${ctx}] Applying sample namespace, helloworld Service and sleep client"
    kubectl --context "${ctx}" apply -f "${SCRIPT_DIR}/sample-app.yaml"
  done

  log "[${EAST_CTX}] helloworld v1"
  kubectl --context "${EAST_CTX}" -n sample apply -f "${SCRIPT_DIR}/helloworld-v1.yaml"

  log "[${WEST_CTX}] helloworld v2"
  kubectl --context "${WEST_CTX}" -n sample apply -f "${SCRIPT_DIR}/helloworld-v2.yaml"

  kubectl --context "${EAST_CTX}" -n sample rollout status deploy/helloworld-v1 --timeout=180s
  kubectl --context "${WEST_CTX}" -n sample rollout status deploy/helloworld-v2 --timeout=180s
  kubectl --context "${EAST_CTX}" -n sample rollout status deploy/sleep --timeout=180s
  kubectl --context "${WEST_CTX}" -n sample rollout status deploy/sleep --timeout=180s
}

############################################
main() {
  create_clusters
  install_metallb
  generate_ca
  install_cacerts
  install_istio
  link_clusters
  install_observability
  deploy_sample

  log "Done. Run ./test.sh to verify cross-cluster traffic."
}

main "$@"
