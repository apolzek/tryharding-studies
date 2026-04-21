---
title: Flux CD — GitOps nativo por Kustomize / Helm
tags: [cncf, graduated, gitops, flux, kustomize, helm]
status: stable
---

## Flux CD (CNCF Graduated)

**O que é:** conjunto de controllers (source, kustomize, helm, notification, image) que sincroniza Kubernetes com Git. Diferente do Argo CD, não tem UI própria — vive no kubectl / Kubernetes Dashboard / Grafana.

**Flux vs Argo CD (resumão honesto):**

| | Flux | Argo CD |
|-|------|---------|
| UI | kubectl + CLI | Web UI rica |
| Modelo | CRDs granulares (GitRepository, Kustomization, HelmRelease) | Application |
| Multi-tenant | Tenants via namespace + RBAC | AppProject |
| Image updates | Controller dedicado (ImagePolicy) | Via pipeline externo |
| Peso | Leve, composable | Monolítico com extras |

Use Flux quando: time gosta de CLI, fluxo focado em Kustomize/Helm, quer tudo via CRDs.

**Quando usar (SRE day-to-day):**

- GitOps puro sem dependência de UI externa (só `flux get`, `kubectl get gitrepo`).
- Auto image updates (imagem nova no registry → PR automático no Git → apply).
- Progressive delivery via **Flagger** (canário/blue-green).

### Cenário real

*"Tenho um Kustomize base + overlays por ambiente no GitHub. Quero que o cluster aplique automaticamente e reconcilie a cada 2 minutos."*

### Reproducing

```bash
cd content/044/flux

# 1. Cluster
kind create cluster --config kind.yaml

# 2. Instala Flux components
kubectl apply -f https://github.com/fluxcd/flux2/releases/download/v2.4.0/install.yaml
kubectl -n flux-system wait --for=condition=available --timeout=5m deploy --all

# 3. Aponta Flux para um repo público (podinfo demo)
kubectl apply -f gitrepo.yaml

# 4. Observa reconciliação
kubectl -n flux-system get gitrepositories,kustomizations
kubectl get deploy,svc -n default | grep podinfo
```

### Status via CRDs

```bash
# tudo que Flux conhece
kubectl get gitrepositories,kustomizations,helmrepositories,helmreleases -A

# último sync de cada
kubectl -n flux-system get kustomizations -o custom-columns=NAME:.metadata.name,LAST_SYNC:.status.lastAppliedRevision,READY:.status.conditions[0].status
```

### Cleanup

```bash
kind delete cluster --name flux-poc
```

### Tips de SRE

- **Notification controller**: envia alerts p/ Slack/Teams/webhook quando sync falha. Configure no dia 1 — sem isso, você só descobre quebra pela queixa.
- **ImagePolicy + ImageUpdateAutomation** automatizam bump de imagem no Git. Cuidado: quem revisa a tag nova?
- **Flux multi-tenancy** por namespace + `spec.serviceAccountName` — cada time só mexe no seu namespace.
- `flux trace <kind> <name>` rastreia do recurso até o Git — essencial em incidente.

### References

- https://fluxcd.io/flux/
- https://github.com/stefanprodan/podinfo (app canônica de teste)
