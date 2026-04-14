#!/usr/bin/env bash
set -euo pipefail

NS=monitoring

echo "==> pods"
kubectl -n $NS get pods

echo
echo "==> probes registered"
kubectl -n $NS get probes.monitoring.coreos.com

echo
echo "==> direct blackbox test (http_2xx -> https://www.google.com)"
kubectl -n $NS run bb-curl --rm -it --restart=Never --image=curlimages/curl:8.10.1 -- \
  curl -s "http://blackbox-prometheus-blackbox-exporter.monitoring.svc.cluster.local:9115/probe?target=https://www.google.com&module=http_2xx" \
  | grep -E 'probe_success|probe_http_status_code|probe_duration_seconds '

echo
echo "==> query Prometheus for probe_success (top 10)"
kubectl -n $NS exec -it statefulset/prometheus-kps-prometheus -c prometheus -- \
  wget -qO- 'http://localhost:9090/api/v1/query?query=probe_success' \
  | head -c 2000
echo
