---
title: Knative Serving — serverless em Kubernetes (scale-to-zero)
tags: [cncf, graduated, knative, serverless, kpa, faas]
status: stable
---

## Knative (CNCF Graduated)

**O que é:** "serverless on Kubernetes". Duas peças principais: **Serving** (HTTP stateless workloads com scale-to-zero, revisions, traffic split) e **Eventing** (pipelines de eventos).

**Quando usar (SRE day-to-day):**

- Services com tráfego burst-y onde scale-to-zero economiza muito (HTTP APIs, webhooks).
- Canário automático via revisions + traffic percentage (built-in).
- Plataforma interna de "sobe seu contêiner e pronto" (PaaS simples).

**Quando NÃO usar:**

- Long-running / stateful — Knative Serving é stateless + HTTP-only.
- Cold start é inaceitável (ex: API de latency crítica P50 <50ms). Use `minScale: 1`.
- Workloads que não falam HTTP (Knative só aceita L7 HTTP).

### Cenário real

*"Quero que meu endpoint `hello` durma quando não tem tráfego (custo zero) e escale até 5 replicas automaticamente quando chegar carga."*

### Reproducing

```bash
cd content/044/knative

# 1. Cluster
kind create cluster --config kind.yaml

# 2. Install Knative Serving CRDs + core
kubectl apply -f https://github.com/knative/serving/releases/download/knative-v1.16.0/serving-crds.yaml
kubectl apply -f https://github.com/knative/serving/releases/download/knative-v1.16.0/serving-core.yaml

# 3. Networking layer — Kourier é o mais leve
kubectl apply -f https://github.com/knative/net-kourier/releases/download/knative-v1.16.0/kourier.yaml

# configura Knative para usar Kourier
kubectl patch configmap/config-network -n knative-serving --type merge \
  -p '{"data":{"ingress-class":"kourier.ingress.networking.knative.dev"}}'

# configura DNS mágico (sslip.io) para POC
kubectl apply -f https://github.com/knative/serving/releases/download/knative-v1.16.0/serving-default-domain.yaml

kubectl -n knative-serving wait --for=condition=available --timeout=5m deploy --all

# 4. Service
kubectl apply -f service.yaml

# 5. Espera ficar pronto
kubectl wait ksvc hello --for=condition=Ready --timeout=5m
kubectl get ksvc hello
```

### Testando scale-to-zero

```bash
# URL do serviço
URL=$(kubectl get ksvc hello -o jsonpath='{.status.url}')
echo $URL

# 1 request
curl $URL

# pods: deve ter 1. Espere ~60s sem tráfego → pods vão p/ 0
kubectl get pods -l serving.knative.dev/service=hello -w

# Traga de volta com um request (cold start)
curl $URL
```

### Traffic split entre revisions (canário)

```yaml
# Atualize service.yaml mudando TARGET, apply e veja:
spec:
  traffic:
    - revisionName: hello-00001
      percent: 80
    - revisionName: hello-00002
      percent: 20
```

### Cleanup

```bash
kind delete cluster --name knative-poc
```

### Tips de SRE

- **Cold start**: `minScale: 1` se SLO não tolera. Pague o custo do pod sempre vivo.
- **Concurrency target** (`autoscaling.knative.dev/target`): quantos requests simultâneos por pod antes de escalar. 50 é um bom ponto de partida; calibre com carga real.
- **KPA vs HPA**: Knative Pod Autoscaler reage em segundos (via request rate). HPA (métrica CPU) reage em 30-60s. KPA ganha para workloads HTTP.
- **Revisions imutáveis** — cada spec muda = revision nova. Rollback é trocar traffic percentage.
- **Net layers**: Kourier (leve, Envoy), Contour, Istio. Escolha por stack existente.
- Em prod: use **cert-manager** + domain real + Kourier com TLS.

### References

- https://knative.dev/docs/
- https://knative.dev/docs/serving/autoscaling/
