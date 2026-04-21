---
title: Dragonfly — P2P image & file distribution para clusters grandes
tags: [cncf, graduated, dragonfly, p2p, registry, image-distribution]
status: stable
---

## Dragonfly (CNCF Graduated)

**O que é:** sistema de distribuição de arquivos/imagens container **baseado em P2P**. Pegue um cluster com 1000 nós puxando a mesma imagem de 2GB do registry — isso é 2TB de egress cobrado pelo registry. Com Dragonfly, 1 nó puxa do registry, os outros 999 puxam **dos peers** — tráfego upstream cai drasticamente.

**Quando usar (SRE day-to-day):**

- Clusters grandes (>100 nós) onde pull simultâneo de imagem satura registry/network.
- Deploys em janela de mudança (todos os pods de um DaemonSet puxam ao mesmo tempo).
- Ambientes com link upstream caro ou limitado (edge, IoT, regiões remotas).
- **Autenticação preservada** — Dragonfly faz HEAD authenticado no registry e replica só para peers do mesmo tenant.

**Quando NÃO usar:**

- Cluster pequeno — o overhead de rodar Scheduler+Manager+DaemonSet de peers é maior que a economia.
- Imagens diferentes em cada nó (sem reutilização).

### Cenário real

*"Meu cluster EKS de 500 nós puxa a mesma imagem de 1GB a cada deploy. O bill de egress do registry/ECR é absurdo. Quero que só alguns nós puxem do registry e o resto pegue de vizinhos."*

### Reproducing

```bash
cd content/044/dragonfly

# 1. Cluster
kind create cluster --config kind.yaml

# 2. Install Dragonfly via Helm (ATENÇÃO: deps Bitnami)
helm repo add dragonfly https://dragonflyoss.github.io/helm-charts/
helm install dragonfly dragonfly/dragonfly --version 1.2.10 \
  -n dragonfly-system --create-namespace --wait --timeout 10m

kubectl -n dragonfly-system get pods
```

> ⚠️ **Breaking change Bitnami (ago/2025)**: o chart usa Bitnami MySQL/Redis como subchart. A Bitnami moveu as imagens para `bitnamilegacy/*` ou `bitnamisecure/*` (pago). Para este POC funcionar, sobrescreva as imagens:
> ```bash
> helm install dragonfly dragonfly/dragonfly -n dragonfly-system --create-namespace \
>   --set mysql.image.registry=docker.io --set mysql.image.repository=bitnamilegacy/mysql \
>   --set redis.image.registry=docker.io --set redis.image.repository=bitnamilegacy/redis
> ```
> ou troque para Postgres nativo / KeyDB via values customizado.

Deve rodar:
- **Manager**: controle/UI (porta 8080)
- **Scheduler**: decide de quem cada peer pega
- **Seed peer**: primeiro a puxar do origin
- **Dfdaemon** (DaemonSet): agente em cada nó, faz proxy do pull

### Como containerd usa

```bash
# Dragonfly registra-se como mirror do containerd via plugin
# containerd pull com config mirror:
#   [plugins."io.containerd.grpc.v1.cri".registry.mirrors."docker.io"]
#     endpoint = ["http://127.0.0.1:65001"]
#
# Primeira vez: seed peer pull de docker.io, serve para o cluster.
# Segunda vez: peers puxam entre si.
```

### Cleanup

```bash
kind delete cluster --name dragonfly-poc
```

### Tips de SRE

- **Storage**: peers fazem cache em disco local. Prevê 20-50% da maior imagem × nós. SSD.
- **Preheat**: API `/api/v1/jobs/preheats` — antes do deploy, instrui o cluster a pré-puxar a imagem. Zero cold start.
- **Métricas**: Dragonfly expõe Prometheus. Alerte em `piece_download_failed_total` > threshold.
- **Nydus + Dragonfly**: Nydus é formato de imagem "on-demand" (só puxa os chunks usados). Combina bem com Dragonfly P2P = deploys sub-segundo em imagem de GBs.
- Compatível com Harbor (P2P preheat integrado na Harbor v2.8+).

### References

- https://d7y.io/
- https://github.com/dragonflyoss/helm-charts
