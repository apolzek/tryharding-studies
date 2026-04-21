---
title: CNCF Landscape Tour — 42 projetos com POCs práticos para SRE
tags: [cncf, graduated, incubating, sre, kubernetes, docker-compose, kind]
status: stable
---

## CNCF Landscape Tour

Curadoria de 42 projetos **CNCF Graduated + Incubating** com POC rodável em `docker compose` ou `kind` + README no formato "dia a dia de SRE" (o que é, quando usar, quando NÃO usar, cenário real, comandos, tips operacionais).

Cada pasta é autocontida e faz cleanup entre POCs — sem conflito de portas, clusters ou volumes.

### GRADUATED (25)

| # | Projeto | Tipo | Pasta |
|--:|---------|------|-------|
|  1 | Prometheus            | Observability (metrics)       | [prometheus](./prometheus) |
|  2 | Fluentd               | Observability (logs)          | [fluentd](./fluentd) |
|  3 | Jaeger                | Observability (tracing)       | [jaeger](./jaeger) |
|  4 | Helm                  | Package manager               | [helm](./helm) |
|  5 | Argo CD               | GitOps (UI-centric)           | [argocd](./argocd) |
|  6 | Flux                  | GitOps (CRD-centric)          | [flux](./flux) |
|  7 | cert-manager          | TLS automation                | [cert-manager](./cert-manager) |
|  8 | Harbor                | Registry OCI + scan + sign    | [harbor](./harbor) |
|  9 | Envoy                 | L7 proxy                      | [envoy](./envoy) |
| 10 | Linkerd               | Service mesh (leve)           | [linkerd](./linkerd) |
| 11 | Istio                 | Service mesh (completo)       | [istio](./istio) |
| 12 | Cilium                | CNI eBPF + Hubble             | [cilium](./cilium) |
| 13 | CoreDNS               | DNS plugável                  | [coredns](./coredns) |
| 14 | etcd                  | KV distribuído (Raft)         | [etcd](./etcd) |
| 15 | KEDA                  | Event-driven autoscaling      | [keda](./keda) |
| 16 | Kyverno               | Policy (YAML)                 | [kyverno](./kyverno) |
| 17 | OPA                   | Policy (Rego, general purpose)| [opa](./opa) |
| 18 | Falco                 | Runtime security (syscalls)   | [falco](./falco) |
| 19 | Knative               | Serverless (scale-to-zero)    | [knative](./knative) |
| 20 | Dapr                  | Distributed app runtime       | [dapr](./dapr) |
| 21 | Crossplane            | IaC via CRD                   | [crossplane](./crossplane) |
| 22 | Vitess                | MySQL sharding                | [vitess](./vitess) |
| 23 | SPIRE                 | Workload identity (SPIFFE)    | [spire](./spire) |
| 24 | Dragonfly             | P2P image distribution        | [dragonfly](./dragonfly) |
| 25 | Rook                  | Ceph operator (storage)       | [rook](./rook) |

### INCUBATING (17)

| # | Projeto | Tipo | Pasta |
|--:|---------|------|-------|
| 26 | Thanos                | Long-term Prometheus          | [thanos](./thanos) |
| 27 | Cortex                | Multi-tenant Prometheus       | [cortex](./cortex) |
| 28 | Longhorn              | Block storage cloud-native    | [longhorn](./longhorn) |
| 29 | Backstage             | Developer portal              | [backstage](./backstage) |
| 30 | NATS                  | Messaging leve + JetStream    | [nats](./nats) |
| 31 | Chaos Mesh            | Chaos engineering             | [chaos-mesh](./chaos-mesh) |
| 32 | Contour               | Ingress controller (Envoy)    | [contour](./contour) |
| 33 | Emissary-Ingress      | API Gateway (Envoy)           | [emissary](./emissary) |
| 34 | OpenCost              | FinOps / cost allocation      | [opencost](./opencost) |
| 35 | Kubescape             | Security scanner              | [kubescape](./kubescape) |
| 36 | Strimzi               | Kafka operator                | [strimzi](./strimzi) |
| 37 | OpenTelemetry         | Collector unificado           | [opentelemetry](./opentelemetry) |
| 38 | KubeVirt              | VM em k8s                     | [kubevirt](./kubevirt) |
| 39 | Litmus                | Chaos engineering + ChaosHub  | [litmus](./litmus) |
| 40 | Artifact Hub          | Catálogo de charts/operators  | [artifact-hub](./artifact-hub) |
| 41 | OpenFeature           | Feature flags (flagd)         | [openfeature](./openfeature) |
| 42 | Kubeflow              | ML platform                   | [kubeflow](./kubeflow) |

### Bonus

| # | Projeto | Tipo | Pasta |
|--:|---------|------|-------|
| 43 | gRPC                  | RPC protocol (Incubating)     | [grpc](./grpc) |

### Cenas reais cobertas por área

- **Observabilidade**: Prometheus → Thanos/Cortex (long-term + multi-tenant), OpenTelemetry (coletor), Jaeger, Fluentd.
- **Config/Policy**: Kyverno, OPA, cert-manager, OpenFeature, Artifact Hub.
- **Networking/Mesh**: Envoy, Linkerd, Istio, Cilium, Contour, Emissary, CoreDNS.
- **Security**: Falco, Kubescape, SPIRE, Harbor.
- **GitOps / Platform**: Argo CD, Flux, Crossplane, Backstage, Helm.
- **Scaling / Serverless**: KEDA, Knative, Dapr.
- **Storage / DB**: Rook, Longhorn, etcd, Vitess.
- **Messaging / Streaming**: NATS, Strimzi (Kafka), gRPC.
- **Chaos / Resiliência**: Chaos Mesh, Litmus.
- **Cost / FinOps**: OpenCost.
- **ML / Virtual**: Kubeflow, KubeVirt.
- **Distribuição de imagem**: Dragonfly.

### Pré-requisitos gerais

- Docker Engine 24+ e Docker Compose v2
- kind 0.24+ e kubectl matching
- Helm 3.14+
- ~8 GB RAM livre para as POCs kind (Kubeflow, Vitess, Rook, KubeVirt precisam MUITO mais)
- Kernel Linux 5.8+ (Cilium, Falco modern eBPF)

### Patterns usados

1. **kind config** isolado por POC (`kind.yaml`), nome `<proj>-poc` para evitar colisão.
2. **docker-compose** com `container_name: cncf-<proj>` para fácil identificação e cleanup.
3. **Cleanup obrigatório** (`docker compose down -v` / `kind delete cluster`) ao final de cada POC.
4. **README com estrutura fixa**: O que é / Quando usar / Quando NÃO usar / Cenário real / Reproducing / Cleanup / Tips SRE / References.

### Notas de compatibilidade

- **Bitnami images quebraram (ago/2025)**: charts com deps Bitnami (Dragonfly redis/mysql, Longhorn, Strimzi, OpenCost prom) podem precisar de overrides para `bitnamilegacy/*` ou providers alternativos. Cada POC afetado tem nota no README.
- **Rook, KubeVirt, Kubeflow**: rodam em kind com limitações (raw block devices, nested virt, 20GB+ RAM). Em prod, nós dedicados.
- **Falco modern_ebpf**: precisa de kernel host com BPF ativado; docker-desktop pode bloquear.

### References

- https://www.cncf.io/projects/
- https://landscape.cncf.io/
