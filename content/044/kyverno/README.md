---
title: Kyverno — policy engine nativo do Kubernetes (sem Rego)
tags: [cncf, graduated, kyverno, policy, admission-controller, security]
status: stable
---

## Kyverno (CNCF Graduated)

**O que é:** policy engine para Kubernetes que roda como admission controller. Diferente do OPA/Gatekeeper, **não usa Rego** — policies são YAML com o mesmo schema de manifests. Faz: **validate** (rejeita), **mutate** (edita), **generate** (cria outro recurso), **verifyImages** (signature).

**Kyverno vs OPA/Gatekeeper (decisão rápida):**

| | Kyverno | Gatekeeper |
|-|---------|------------|
| Linguagem | YAML (pattern-match estilo k8s) | Rego |
| Curva aprendizado | baixa | alta |
| Features | validate + mutate + generate + verifyImages | validate + mutate (v3.12+) |
| Performance | Excelente em k8s | Excelente; Rego um pouco mais pesado |
| Fora do k8s | Não (100% k8s) | Sim (OPA é general-purpose) |

Se política roda só em k8s → Kyverno. Se precisa reusar a mesma política em API gateway, Terraform, CI → OPA.

**Quando usar (SRE day-to-day):**

- Bloquear `:latest` (proibido em prod — deploy não-reproducível).
- Forçar `resources.requests/limits` (senão o pod estoura nó).
- Exigir labels obrigatórios (`team`, `cost-center`).
- Forçar `readOnlyRootFilesystem: true`, `runAsNonRoot: true` (security baseline).
- **Mutate** automático: injetar sidecar, adicionar tolerations, default limits.
- **Generate**: criar NetworkPolicy default por namespace novo.
- **verifyImages** com Cosign: só imagens assinadas podem rodar.

### Cenário real

*"Quero proibir deploys sem resources limits e sem tag de imagem explícita no cluster inteiro."*

### Reproducing

```bash
cd content/044/kyverno

# 1. Cluster
kind create cluster --config kind.yaml

# 2. Install
helm repo add kyverno https://kyverno.github.io/kyverno/
helm repo update
helm install kyverno kyverno/kyverno --version 3.3.2 \
  -n kyverno --create-namespace --wait

# 3. Aplica policies
kubectl apply -f policy.yaml
kubectl get clusterpolicies

# 4. Tentativa que DEVE falhar (latest + sem resources)
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata: { name: bad }
spec:
  containers: [ { name: c, image: nginx:latest } ]
EOF
# → Error from server: admission webhook denied

# 5. Tentativa que passa
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata: { name: good }
spec:
  containers:
    - name: c
      image: nginx:1.27
      resources:
        requests: { cpu: 50m, memory: 64Mi }
        limits:   { memory: 128Mi }
EOF
```

### Cleanup

```bash
kind delete cluster --name kyverno-poc
```

### Tips de SRE

- **Enforce vs Audit**: comece com `validationFailureAction: Audit` — só loga violações. Migre para `Enforce` depois de limpar o cluster.
- **Exceptions**: `PolicyException` CRD dispensa namespaces/workloads específicos — docs-as-code da exceção.
- **PolicyReport**: resultado de audits fica em `kubectl get polr,cpolr -A`. Exporte para Grafana.
- **Catálogo oficial**: [kyverno.io/policies](https://kyverno.io/policies/) — 150+ políticas prontas (PSS baseline, EKS, GKE, OpenShift).
- **Background scan**: Kyverno também varre recursos **já existentes**, não só admission. Encontra o passivo.
- Em clusters grandes, `admissionController.replicas=3` com `podAntiAffinity` — evita SPOF.

### References

- https://kyverno.io/docs/
- https://kyverno.io/policies/
