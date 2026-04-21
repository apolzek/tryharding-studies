---
title: Helm — package manager do Kubernetes
tags: [cncf, graduated, helm, packaging, gitops-friendly]
status: stable
---

## Helm (CNCF Graduated)

**O que é:** gerenciador de pacotes para Kubernetes. Um **chart** é um template (Go template) de manifests com `values.yaml` como entrada. `helm install` renderiza e aplica; guarda revisão para rollback.

**Quando usar (SRE day-to-day):**

- Instalar stacks de terceiros (ingress, cert-manager, prometheus, argocd, etc) sem escrever manifest do zero.
- Versionar valores por ambiente (`values-dev.yaml`, `values-prod.yaml`).
- Rollback rápido de release ruim (`helm rollback <release> <rev>`).
- Usar como source para ArgoCD/Flux — eles renderizam helm antes de aplicar.

**Quando NÃO usar (ou ter cuidado):**

- Charts complexos com muito `if/else` no template viram inferno. Considere **kustomize** para overlays simples ou **Helmfile** para orquestrar múltiplos releases.
- `helm install` direto em prod sem GitOps = drift. Prefira Argo Application / Flux HelmRelease apontando para o chart.
- Secrets em `values.yaml` commitados — use `helm-secrets` ou sealed-secrets / external-secrets.

### Cenário real

*"Preciso subir o nginx-ingress em 3 ambientes (dev/stg/prod) com replica count e recursos diferentes, e conseguir dar rollback se quebrar o cluster."*

Este POC instala nginx-ingress via Helm, faz upgrade com valores customizados e demonstra rollback.

### Reproducing

```bash
cd content/044/helm

# 1. Cluster
kind create cluster --config kind.yaml
kubectl cluster-info --context kind-helm-poc

# 2. Cria chart local (o `helm create` scaffolda um chart nginx funcional)
helm create demo-app

# 3. Install
helm install web ./demo-app --wait

# 4. Upgrade com values de prod (2 replicas + resource limits)
helm upgrade web ./demo-app -f values-prod.yaml --wait

# 5. History (cada upgrade = nova revisão)
helm history web

# 6. Rollback para revisão 1
helm rollback web 1 --wait
helm history web

# 7. Valida
kubectl get deploy,svc -l app.kubernetes.io/instance=web
```

### Cleanup

```bash
helm uninstall web
kind delete cluster --name helm-poc
```

### Tips de SRE

- `helm install --dry-run --debug` antes de qualquer upgrade arriscado — vê o manifest final.
- `helm diff upgrade` (plugin) mostra diff antes de aplicar — essencial em review de PR.
- Pin **sempre** a versão do chart (`--version`). Charts mudam — seu ambiente não pode surpresar.
- Em GitOps, o chart fica no Git via `Chart.yaml` + `values.yaml`; deixe o Argo/Flux aplicar, não `helm install` manual.
- Release names e namespaces: `-n <ns>` sempre. `default` é o começo da bagunça.

### References

- https://helm.sh/docs/
- https://artifacthub.io/ (biblioteca central de charts — é um POC separado aqui)
