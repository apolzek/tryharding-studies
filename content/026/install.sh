#!/usr/bin/env bash
set -euo pipefail

CLUSTER=blackbox-lab
NS=monitoring

echo "==> creating kind cluster (k8s v1.33.1)"
kind create cluster --config kind-config.yaml

echo "==> adding helm repos"
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

echo "==> namespace"
kubectl create ns $NS --dry-run=client -o yaml | kubectl apply -f -

echo "==> installing kube-prometheus-stack (ships Prometheus Operator + Grafana)"
helm upgrade --install kps prometheus-community/kube-prometheus-stack \
  -n $NS \
  -f values-kube-prometheus-stack.yaml \
  --wait --timeout 10m

echo "==> installing prometheus-blackbox-exporter"
helm upgrade --install blackbox prometheus-community/prometheus-blackbox-exporter \
  -n $NS \
  -f values-blackbox.yaml \
  --wait --timeout 5m

echo "==> applying Probe CRDs (targets)"
kubectl apply -f probes.yaml

echo "==> installing blackbox Grafana dashboard (sidecar pickup)"
kubectl apply -f blackbox-dashboard-cm.yaml

echo
echo "Grafana NodePort:   http://localhost:30300  (admin / admin)"
echo "Prometheus:         http://localhost:30090"
echo
echo "To port-forward Grafana on :3000 instead, run:"
echo "  kubectl -n $NS port-forward svc/kps-grafana 3000:80"
echo
echo "Blackbox dashboard: http://127.0.0.1:3000/d/xtkCtBkiz/blackbox-exporter"
echo "(same dashboard is also reachable via the NodePort URL above)"
