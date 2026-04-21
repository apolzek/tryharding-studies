---
title: Contour — Ingress controller sobre Envoy com HTTPProxy
tags: [cncf, incubating, contour, ingress, envoy, gateway-api]
status: stable
---

## Contour (CNCF Incubating)

**O que é:** Ingress controller para Kubernetes baseado em **Envoy**. Ao invés de depender só do `Ingress` limitado, oferece CRD `HTTPProxy` com features ricas: weighted routing, retries, timeouts, rate-limit, TLS termination avançado, delegation multi-tenant. Também fala **Gateway API** (sucessor do Ingress).

**Contour vs nginx-ingress vs Istio Gateway:**

| | Contour | nginx-ingress | Istio Gateway |
|-|---------|---------------|---------------|
| Proxy | Envoy | nginx | Envoy |
| CRD | HTTPProxy + Gateway API | Ingress + annotations | Gateway + VirtualService |
| Observabilidade | Prometheus nativo | Precisa módulo | Prometheus+tracing |
| Canário | Built-in weights | Annotation | VirtualService |
| Peso | Médio | Leve | Precisa mesh completo |

**Quando usar (SRE day-to-day):**

- Ingress com features de L7 real (weighted canary sem Istio).
- Multi-tenant com **HTTPProxy delegation** — team-a só mexe em `/api/team-a/*` do virtualhost.
- Gateway API sem adotar service mesh completo.
- Clusters que já têm Envoy em outro lugar (consistência).

### Cenário real

*"Quero um Ingress Controller que aceite retries automáticos e timeouts declarativos por rota. nginx-ingress faz com annotations feias; quero CRD."*

### Reproducing

```bash
cd content/044/contour

# 1. Cluster (porta 30080 exposta)
kind create cluster --config kind.yaml

# 2. Install Contour
kubectl apply -f https://raw.githubusercontent.com/projectcontour/contour/v1.30.1/examples/render/contour.yaml

# patch: service Envoy para NodePort 30080 (kind mapeou p/ host)
kubectl -n projectcontour patch svc envoy -p \
  '{"spec":{"type":"NodePort","ports":[{"name":"http","port":80,"nodePort":30080}]}}'

kubectl -n projectcontour wait --for=condition=available --timeout=5m deploy --all

# 3. App + HTTPProxy
kubectl apply -f httpproxy.yaml
kubectl wait --for=condition=available deploy/web --timeout=2m

# 4. Teste
curl -s --resolve web.local:30080:127.0.0.1 http://web.local:30080/ | head -1
# → <!DOCTYPE html> (nginx)
```

### Cleanup

```bash
kind delete cluster --name contour-poc
```

### Tips de SRE

- **HTTPProxy delegation**: root proxy define fqdn + tls, delega prefixos para HTTPProxy em namespaces de times — cada time só mexe no seu. Multi-tenant real.
- **Visibilidade**: `kubectl get httpproxy -A -o jsonpath='{range .items[*]}{.metadata.name} {.status.currentStatus}{"\n"}{end}'`
- **Retries** — cuidado com `retryOn: 5xx` em POST (idempotência).
- **TLS**: `spec.virtualhost.tls.secretName` — vem do cert-manager.
- **Gateway API**: migração gradual do Ingress. Contour implementa `Gateway`/`HTTPRoute` v1.

### References

- https://projectcontour.io/docs/
- https://projectcontour.io/docs/v1.30/config/fundamentals/
