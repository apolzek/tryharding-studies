---
title: Istio — service mesh completo (L7 routing, mTLS, telemetria)
tags: [cncf, graduated, service-mesh, istio, envoy]
status: stable
---

## Istio (CNCF Graduated)

**O que é:** service mesh maduro construído sobre Envoy como data plane. Control plane unificado (`istiod`) fornece: descoberta de serviço, roteamento L7 (VirtualService), políticas (DestinationRule, PeerAuthentication), observabilidade e segurança (mTLS automático + SPIFFE).

**Quando usar (SRE day-to-day):**

- **Canário/blue-green** com weight por versão (VirtualService).
- **mTLS STRICT** cluster-wide — requer `PeerAuthentication` mode: STRICT.
- **Fault injection** para chaos test (`httpFault: { delay / abort }` em VirtualService).
- **Rate limit + authz** em sidecar, sem mexer na app.
- **Kiali + Jaeger + Grafana** pré-integrados para mapa de serviços e traces.

**Quando NÃO usar:**

- Overhead operacional real — sidecar/ambient-mesh, updates, troubleshoot de Envoy config. Se só precisa de mTLS e métricas simples, **Linkerd** é mais leve.
- Clusters pequenos (<10 serviços) geralmente não justificam o custo cognitivo.

### Cenário real

*"Quero fazer deploy canário com 10% do tráfego na v2 do microsserviço `reviews` e monitorar error rate antes de bump pra 50/50."*

Este POC instala Istio com profile `demo`, sobe a clássica app `bookinfo` e aplica um VirtualService de traffic split 90/10.

### Reproducing

```bash
cd content/044/istio

# 1. Cluster
kind create cluster --config kind.yaml

# 2. istioctl
curl -L https://istio.io/downloadIstio | ISTIO_VERSION=1.24.0 sh -
export PATH=$PWD/istio-1.24.0/bin:$PATH

# 3. Install (profile demo = control plane + ingress gateway + egress + telemetria)
istioctl install --set profile=demo -y
kubectl -n istio-system wait --for=condition=available --timeout=5m deploy --all

# 4. Bookinfo demo app
kubectl label namespace default istio-injection=enabled
kubectl apply -f istio-1.24.0/samples/bookinfo/platform/kube/bookinfo.yaml
kubectl apply -f istio-1.24.0/samples/bookinfo/networking/bookinfo-gateway.yaml
kubectl wait --for=condition=ready --timeout=5m pod --all

# 5. Traffic split 90/10
kubectl apply -f traffic-shift.yaml

# 6. Teste: várias requests devem sair ~90% em v1, ~10% em v2
for i in $(seq 1 20); do
  kubectl exec deploy/ratings-v1 -c ratings -- curl -s http://reviews:9080/reviews/0 | grep -o '"podname":"[^"]*"'
done
```

### Observabilidade (Kiali + Jaeger)

```bash
# addons que vêm no samples/
kubectl apply -f istio-1.24.0/samples/addons/
kubectl -n istio-system rollout status deployment kiali
istioctl dashboard kiali   # abre o browser
```

### Cleanup

```bash
kind delete cluster --name istio-poc
rm -rf istio-1.24.0
```

### Tips de SRE

- **mTLS STRICT cluster-wide**: `kubectl apply -f -` com um `PeerAuthentication mode: STRICT` no namespace `istio-system`. Depois que estiver ok, propague.
- **Sidecar scope**: por default cada sidecar conhece todos os services. Isso explode em clusters grandes — use `Sidecar` resource para limitar.
- **Ambient mode** (sem sidecar): Istio 1.22+ suporta L4 por ztunnel + L7 por waypoint proxy. Menos overhead mas menos features por enquanto.
- **Canário**: sempre paire traffic split com `Argo Rollouts` ou `Flagger` que automatizam rollback por SLO.
- `istioctl analyze` antes de aplicar mudanças — pega conflitos de config.

### References

- https://istio.io/latest/docs/
- https://istio.io/latest/docs/tasks/traffic-management/traffic-shifting/
