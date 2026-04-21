---
title: Argo CD — GitOps declarativo para Kubernetes
tags: [cncf, graduated, gitops, argocd, continuous-deployment]
status: stable
---

## Argo CD (CNCF Graduated)

**O que é:** controller GitOps que mantém o estado do cluster sincronizado com um repositório Git. Você declara uma `Application` apontando para um repo+path e o Argo faz drift detection + sync automático (ou manual).

**Quando usar (SRE day-to-day):**

- **Deploys auditáveis** — PR merge = deploy. Quem deployou? `git log`. Revert? `git revert`.
- **Cluster as Git** — stand up de DR é só apontar um novo cluster para o mesmo repo.
- **Multi-cluster**: um Argo gerencia N clusters (cluster hub/spokes).
- **App of Apps** pattern — uma Application raiz aponta para um path de Applications filhas, versiona toda a topologia.

**Quando NÃO usar:**

- Se o pipeline é "kubectl apply manual" ocasional, Argo é overkill.
- Secrets no Git cru = não. Combine com sealed-secrets / external-secrets / sops.
- Manifests gerados dinamicamente (ex: Jenkins que faz sed) não se encaixam — converta para Helm/Kustomize.

### Cenário real

*"Quero que cada merge no branch main do repo `infra-apps` vire deploy automático no cluster de prod, e se alguém mexer no cluster direto (kubectl edit), o Argo detecta e reverte."*

Este POC instala Argo CD e cria uma `Application` apontando para o repo público `argocd-example-apps` com sync automático + selfHeal.

### Reproducing

```bash
cd content/044/argocd

# 1. Cluster
kind create cluster --config kind.yaml

# 2. Instala Argo CD
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/v2.12.4/manifests/install.yaml

# 3. Espera ficar pronto
kubectl -n argocd wait --for=condition=available --timeout=5m deploy --all

# 4. Cria Application (aponta p/ repo público)
kubectl apply -f application.yaml

# 5. Observa sync
kubectl -n argocd get applications -w
# em outro terminal:
kubectl -n guestbook get pods
```

### Acessando a UI

```bash
# Password do admin (inicial)
kubectl -n argocd get secret argocd-initial-admin-secret \
  -o jsonpath='{.data.password}' | base64 -d && echo

# Port-forward
kubectl -n argocd port-forward svc/argocd-server 8080:443
# https://localhost:8080 (user: admin)
```

### Testando selfHeal

```bash
# "Drift" manual
kubectl -n guestbook scale deploy guestbook-ui --replicas=5
# Argo vai reverter em ~30s (porque Application tem selfHeal: true)
kubectl -n guestbook get deploy guestbook-ui -w
```

### Cleanup

```bash
kind delete cluster --name argocd-poc
```

### Tips de SRE

- **App of Apps**: Application raiz `root-app` apontando para `apps/` — cada subpasta é outra Application. Uma única entrada no Git controla o cluster inteiro.
- `syncPolicy.automated.prune: true` — se você remove do Git, Argo remove do cluster. Sem isso, drift acumula.
- **Health checks customizados**: recursos CRD "desconhecidos" ficam Unknown. Registre health checks Lua em `argocd-cm` ConfigMap.
- **RBAC por projeto** — `AppProject` limita quem pode fazer deploy onde (essencial em multi-team).
- **Webhook do Git** acelera detecção (padrão é polling de 3 min).

### References

- https://argo-cd.readthedocs.io/
- https://github.com/argoproj/argocd-example-apps
