---
title: Emissary-Ingress — API Gateway sobre Envoy (ex-Ambassador)
tags: [cncf, incubating, emissary, ingress, api-gateway, envoy]
status: stable
---

## Emissary-Ingress (CNCF Incubating)

**O que é:** API Gateway Envoy-based feito para microsserviços. Ex-Ambassador. CRDs: `Mapping` (rota), `Host` (virtualhost + TLS), `TLSContext`, `RateLimitService`, `AuthService`, `TCPMapping`. Foco em developer self-service — dev cria Mapping no seu namespace e ganha rota; SRE só define Host central.

**Emissary vs Contour:**

| | Emissary | Contour |
|-|----------|---------|
| Modelo | API Gateway (rotas) | Ingress Controller (HTTPProxy) |
| CRD | Mapping (route-centric) | HTTPProxy (vhost-centric) |
| Extensões | AuthService, RateLimit, Dev Portal | Integração Gateway API |
| Público-alvo | Dev self-service | Platform/SRE-centric |

**Quando usar (SRE day-to-day):**

- Dev team quer self-service de rotas sem SRE tocar em cada PR.
- Features de API Gateway além do Ingress: auth externo, rate-limit, grpc-web, websockets.
- Edge único para 100+ services com routing complexo.

### Cenário real

*"Cada dev do time deveria abrir um PR e ganhar uma rota pública `/team/*` para seu serviço, sem ping no SRE."*

### Reproducing

```bash
cd content/044/emissary

# 1. Cluster
kind create cluster --config kind.yaml

# 2. CRDs + core
kubectl apply -f https://app.getambassador.io/yaml/emissary/3.9.1/emissary-crds.yaml
kubectl wait --timeout=2m --for=condition=available deploy emissary-apiext -n emissary-system

kubectl apply -f https://app.getambassador.io/yaml/emissary/3.9.1/emissary-emissaryns.yaml
kubectl -n emissary wait --for=condition=available --timeout=5m deploy --all

# 3. App + Mapping
kubectl apply -f mapping.yaml

# 4. Test
kubectl -n emissary port-forward svc/emissary-ingress 8080:80 &
sleep 2
curl -s http://localhost:8080/hello/
# → hello from emissary
```

### Cleanup

```bash
kind delete cluster --name emissary-poc
```

### Tips de SRE

- **Host** central: SRE configura `Host` com TLS cert-manager, time só cria `Mapping`.
- **AuthService**: plug OPA/JWT entre edge e services — authz fora da app.
- **Circuit breaker**: `Mapping.circuit_breakers` — protege downstream.
- **Edge Stack** (versão paga): Dev Portal, SSO com Auth0/Keycloak, observability Grafana.
- Troubleshoot: `kubectl -n emissary logs -l service=emissary-ingress` ou `/ambassador/v0/diag/` endpoint interno.

### References

- https://www.getambassador.io/docs/emissary/
- https://www.getambassador.io/docs/emissary/latest/topics/using/intro-mappings
