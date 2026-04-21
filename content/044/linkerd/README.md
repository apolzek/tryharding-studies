---
title: Linkerd — service mesh leve (mTLS + observabilidade)
tags: [cncf, graduated, service-mesh, linkerd, mtls]
status: stable
---

## Linkerd (CNCF Graduated)

**O que é:** service mesh "ultraleve" baseado em um proxy Rust (linkerd2-proxy) feito só para mesh — diferente do Envoy que é general-purpose. Foco em simplicidade operacional.

**Linkerd vs Istio (decisão rápida):**

| | Linkerd | Istio |
|-|---------|-------|
| Proxy | linkerd2-proxy (Rust, pequeno) | Envoy (C++, feature-rich) |
| mTLS | Automático, zero-config | Automático mas com config |
| Features | Mesh essencial | Mesh + gateway + policy + autz rica |
| Setup | `linkerd install`, pronto | Operator + tune |
| Footprint | ~50MB RAM/proxy | ~100-200MB RAM/proxy |
| L7 policy | Authz policies + retries | Rica (VirtualService, DestinationRule) |

Use Linkerd quando: quer mesh funcionando amanhã sem PhD em Envoy.
Use Istio quando: precisa de roteamento L7 complexo, multi-cluster avançado, integrações SPIFFE explícitas.

**Quando usar (SRE day-to-day):**

- **mTLS transparente** entre serviços sem configurar nada.
- **Métricas golden** (RED: rate/error/duration) por request — dashboard Grafana pronto.
- **Retries + timeouts** declarativos (ServiceProfile).
- **Traffic split** canário simples via TrafficSplit CRD.

### Cenário real

*"Preciso de mTLS entre todos meus pods e dashboards de latência por serviço sem escrever Envoy config."*

### Reproducing

```bash
cd content/044/linkerd

# 1. Cluster
kind create cluster --config kind.yaml

# 2. CLI (gera certs EC automaticamente — chart stable exigiria fazer manualmente com `step`)
curl -sL https://run.linkerd.io/install-edge | sh
export PATH=$HOME/.linkerd2/bin:$PATH

# 3. CRDs + control plane
linkerd install --crds | kubectl apply -f -
linkerd install | kubectl apply -f -

kubectl -n linkerd wait --for=condition=available --timeout=5m deploy --all

# 4. Verifica
linkerd check
```

### Injetando mesh na aplicação

```bash
# Cria app demo
kubectl create deploy web --image=nginx
kubectl expose deploy web --port 80

# "Mesh inject": adiciona annotation que faz o admission controller do Linkerd
# injetar o sidecar linkerd2-proxy
kubectl get deploy web -o yaml | linkerd inject - | kubectl apply -f -

# Confirma o sidecar
kubectl get pod -l app=web -o jsonpath='{.items[0].spec.containers[*].name}'
# → nginx linkerd-proxy
```

### Cleanup

```bash
kind delete cluster --name linkerd-poc
```

### Tips de SRE

- **Certs em prod**: use cert-manager emitindo o trust anchor + issuer — Linkerd faz rotação automática do cert de identidade do proxy (24h).
- **linkerd viz** (extensão): Grafana + Prometheus + dashboard de RED metrics. Essencial em prod.
- **ServiceProfile**: define timeouts e retries declarativos por rota — mais simples que Istio VirtualService.
- **Tap** (`linkerd viz tap deploy/web`): vê request por request em tempo real — golden para debug.
- Não misture Linkerd + Istio no mesmo cluster (dois sidecars brigando).

### References

- https://linkerd.io/2/getting-started/
- https://linkerd.io/2/tasks/install-helm/
