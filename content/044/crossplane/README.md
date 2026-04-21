---
title: Crossplane — infra-as-code via CRDs do Kubernetes
tags: [cncf, graduated, crossplane, iac, platform-engineering]
status: stable
---

## Crossplane (CNCF Graduated)

**O que é:** controller plane do Kubernetes estendido para provisionar **recursos externos** (RDS, S3, GKE, Lambda, VMs VSphere, GitHub repos, Datadog monitors...) via CRDs. Você escreve um YAML `RDSInstance` e o Crossplane cria a instância real na AWS — todo o modelo de "apply, reconcile, drift" do k8s vale para cloud resources.

**Crossplane vs Terraform (decisão prática):**

| | Crossplane | Terraform |
|-|-----------|-----------|
| Execução | Controller contínuo (reconcile loop) | CLI (apply / plan pontual) |
| Drift | Corrige automaticamente | Você vê no próximo plan |
| Estado | CRDs no cluster | State file (S3/remote) |
| Linguagem | YAML + Composition | HCL |
| Platform API | Cria CRDs próprios (XRD) p/ dev self-service | Módulos |

Use Crossplane quando quer **platform engineering** (uma CRD `DevDatabase` que dev cria e provê RDS + user + migrations em 1 YAML). TF segue ótimo para infra pontual/operacional.

**Quando usar (SRE day-to-day):**

- Plataforma interna que expõe "primitivas" (`ProductionDatabase`, `DevQueue`) — dev pede em YAML, Crossplane resolve.
- Multi-cloud — mesmo CRD deploya em AWS ou GCP conforme `ProviderConfig`.
- Drift detection contínua (Terraform vê só no plan manual).

### Cenário real

*"Quero que o dev crie `CompositeDatabase` e o Crossplane provisione RDS + security group + Secret no cluster. Sem ir na console AWS."*

### Reproducing

```bash
cd content/044/crossplane

# 1. Cluster
kind create cluster --config kind.yaml

# 2. Install
helm repo add crossplane-stable https://charts.crossplane.io/stable
helm install crossplane crossplane-stable/crossplane \
  --version 1.17.2 -n crossplane-system --create-namespace --wait

# 3. Provider "nop" (dummy — demonstra modelo sem creds reais)
kubectl apply -f provider.yaml

# 4. Espera provider instalar
kubectl wait provider.pkg.crossplane.io/provider-nop \
  --for=condition=healthy --timeout=3m
kubectl get providers

# 5. Cria um recurso nop (simula um "RDSInstance" que não aloca nada real)
cat <<EOF | kubectl apply -f -
apiVersion: nop.crossplane.io/v1alpha1
kind: NopResource
metadata:
  name: db-prod
spec:
  forProvider:
    conditionAfter:
      - time: 5s
        conditionType: Ready
        conditionStatus: "True"
        conditionReason: Available
  providerConfigRef:
    name: default
EOF

# 6. Ver reconciliação
kubectl get nopresource -w
```

### Em produção (ex: AWS)

```bash
# 1. Provider AWS
kubectl apply -f - <<EOF
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata: { name: provider-aws }
spec:
  package: xpkg.upbound.io/upbound/provider-aws-rds:v1.12.0
EOF

# 2. Secret com creds + ProviderConfig
kubectl create secret generic aws-creds -n crossplane-system \
  --from-file=creds=./aws-creds.txt

# 3. RDSInstance via YAML — materializa RDS real
```

### Cleanup

```bash
kind delete cluster --name crossplane-poc
```

### Tips de SRE

- **XRD + Composition** = API pública. Dev pede `MySQLInstance` (simples), Crossplane compõe RDSInstance + Parameter Group + Subnet + Secret (complexo).
- **ProviderConfig por environment** (dev-account vs prod-account) — mesma API, credenciais diferentes.
- **Reconcile interval**: default 10min. Drift é corrigido automaticamente — isso é diferente do Terraform; lembre do time.
- **Deletion policy** (`deletionPolicy: Delete | Orphan`): prod = Orphan (evita `kubectl delete` apagar RDS por engano).
- **Composition Functions** (v1.17+): lógica complexa em Python/Go ao invés de YAML mile-long.

### References

- https://docs.crossplane.io/
- https://marketplace.upbound.io/ (providers)
