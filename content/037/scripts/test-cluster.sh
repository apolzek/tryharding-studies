#!/usr/bin/env bash
set -euo pipefail

ctl() { docker exec k0s-controller k0s kubectl "$@"; }

echo "== waiting for 2 worker nodes Ready =="
for _ in $(seq 1 90); do
  ready=$(ctl get nodes --no-headers 2>/dev/null | awk '$2=="Ready"' | wc -l || echo 0)
  [ "$ready" = "2" ] && break
  sleep 5
done
ctl get nodes -o wide

echo
echo "== preparing hostPath dir on worker-1 =="
docker exec k0s-worker-1 install -d -m 0755 /mnt/data/demo

echo "== releasing stale PV claimRef (if any) =="
ctl patch pv demo-pv-worker-1 --type=json \
  -p='[{"op":"remove","path":"/spec/claimRef"}]' 2>/dev/null || true

echo
echo "== applying sample app =="
docker exec -i k0s-controller k0s kubectl apply -f - < manifests/sample-app.yaml

echo
echo "== waiting for web deployment =="
ctl -n demo rollout status deploy/web --timeout=180s

echo
echo "== demo resources =="
ctl -n demo get pods,svc,pvc,netpol -o wide

echo
echo "== smoke test (curl via cluster DNS from default ns) =="
for attempt in 1 2 3 4 5; do
  out=$(ctl -n default run smoke-$attempt --rm -i --restart=Never --quiet \
    --image=curlimages/curl:8.10.1 --command -- \
    curl -sS --connect-timeout 5 -o /dev/null -w 'HTTP=%{http_code}' http://web.demo.svc/ 2>&1 || true)
  code=$(echo "$out" | grep -oE 'HTTP=[0-9]+' | tail -1 | cut -d= -f2)
  echo "attempt $attempt -> HTTP ${code:-000}"
  [ "$code" = "200" ] && { echo "cluster OK"; exit 0; }
  sleep 5
done
echo "smoke test failed after 5 attempts" >&2
exit 1
