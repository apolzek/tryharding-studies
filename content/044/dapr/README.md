---
title: Dapr — distributed application runtime (building blocks)
tags: [cncf, graduated, dapr, distributed, sidecar, building-blocks]
status: stable
---

## Dapr (CNCF Graduated)

**O que é:** runtime distribuído que expõe "building blocks" via HTTP/gRPC para a app (sidecar pattern): state store, pub/sub, secrets, bindings externos, service invocation, workflows, distributed locks, actors. A **app não importa SDK do Redis/Kafka** — fala com `http://localhost:3500/v1.0/state/statestore` e o Dapr resolve.

**Quando usar (SRE day-to-day):**

- Desacoplar backing stores — trocar Redis por Postgres muda só a Component YAML, app intacta.
- Stack polyglota (Go + Python + .NET + Java) onde cada time reimplementa client — Dapr uniformiza.
- **Resiliência** declarativa (retry, circuit breaker, timeout) por Component, não em cada app.
- **Workflow** (saga, choreography) sem escrever state machine manual.

**Quando NÃO usar:**

- Apps com alto throughput onde o hop via sidecar HTTP é inaceitável.
- Se time tem só 1 stack e está feliz com o driver nativo.

### Cenário real

*"Quero que 5 microsserviços escrevam state sem cada um ter seu client Redis. Amanhã troco Redis por Postgres sem PR em cada repo."*

### Reproducing

```bash
cd content/044/dapr

# 1. Cluster
kind create cluster --config kind.yaml

# 2. Dapr CLI + install
wget -q https://raw.githubusercontent.com/dapr/cli/master/install/install.sh -O - | /bin/bash
dapr init -k --wait --runtime-version 1.14.4

kubectl -n dapr-system get pods

# 3. Redis (statestore backend)
helm repo add bitnami https://charts.bitnami.com/bitnami
helm install redis bitnami/redis --version 20.1.0 \
  --set auth.enabled=false --set architecture=standalone

kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=redis --timeout=3m

# 4. Component
kubectl apply -f statestore.yaml

# 5. App com sidecar injetado
kubectl run client --image=curlimages/curl:8.10.1 \
  --annotations="dapr.io/enabled=true,dapr.io/app-id=client" \
  --command -- sleep 3600
kubectl wait --for=condition=ready pod client --timeout=2m

# 6. Escreve via HTTP na API do Dapr (no localhost do pod)
kubectl exec client -c client -- sh -c 'curl -s -X POST http://localhost:3500/v1.0/state/statestore \
  -H "Content-Type: application/json" \
  -d "[{\"key\": \"order:42\", \"value\": {\"status\":\"paid\"}}]"'

# 7. Lê de volta
kubectl exec client -c client -- curl -s http://localhost:3500/v1.0/state/statestore/order:42
# → {"status":"paid"}
```

### Cleanup

```bash
kind delete cluster --name dapr-poc
```

### Tips de SRE

- **Resiliency policy**: CRD `Resiliency` aplica retry/circuit-breaker por Component. Sem reescrever cliente.
- **mTLS Dapr-to-Dapr**: ligado por default. Checou os certs? `dapr-sentry` gerencia.
- **Observabilidade**: Dapr emite Prometheus + distributed traces via OTel. Plug no Jaeger/Prometheus que você já tem.
- **Component scope**: `scopes:` no YAML restringe qual app pode usar (principle of least privilege).
- **Não é service mesh**: coexiste bem com Linkerd/Istio (sidecar dapr + sidecar mesh no mesmo pod).
- Performance: `http://localhost:3500` é o sidecar — 0.5-1ms overhead. Em latency-critical use gRPC (`dapr.io/app-protocol: grpc`).

### References

- https://docs.dapr.io/
- https://github.com/dapr/quickstarts
