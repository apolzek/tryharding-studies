---
title: Kubeflow — ML platform para Kubernetes (Pipelines + Notebooks + Serving)
tags: [cncf, incubating, kubeflow, ml, mlops, pipelines]
status: stable
---

## Kubeflow (CNCF Incubating)

**O que é:** suíte ML completa para Kubernetes:

- **Kubeflow Pipelines (KFP)** — orquestra ML workflows (ingest → train → eval → deploy) via Argo Workflows por baixo.
- **Notebooks** — Jupyter/VSCode/RStudio self-service por usuário, com GPU.
- **Katib** — HPO (hyperparameter optimization).
- **Training Operator** — `TFJob`, `PyTorchJob`, `MPIJob`, `XGBoostJob` distribuídos.
- **KServe** (ex-KFServing) — model serving escalável (scale-to-zero).
- **Central Dashboard + Multi-tenancy** (profiles/namespaces por team).

**Quando usar (SRE day-to-day):**

- Time de ML que precisa de plataforma padronizada (reproducibilidade, audit).
- Multi-tenant ML (N DS times, recursos isolados, budget per team).
- Pipelines versionados (cada run tem metadata + artifacts em MinIO/S3).
- Serving com autoscaling (KServe + Knative).

**Quando NÃO usar:**

- Cluster pequeno — Kubeflow consome MUITO (60+ deploys).
- Time pequeno sem DS dedicado — MLflow standalone + Airflow pode ser suficiente.

### Cenário real

*"DS team de 10 pessoas. Todos rodam notebook local, compartilham modelo via Slack, deploy no cluster é `kubectl apply` manual. Quero plataforma."*

### Reproducing (Pipelines standalone — menor footprint)

```bash
cd content/044/kubeflow

# 1. Cluster (kind precisa de MUITA RAM; kubeflow completo ~12GB)
kind create cluster --config kind.yaml

# 2. Pipelines standalone (mínimo utilizável, sem auth/dashboard central)
export PIPELINE_VERSION=2.3.0
kubectl apply -k "github.com/kubeflow/pipelines/manifests/kustomize/cluster-scoped-resources?ref=$PIPELINE_VERSION"
kubectl wait --for condition=established --timeout=60s crd/applications.app.k8s.io
kubectl apply -k "github.com/kubeflow/pipelines/manifests/kustomize/env/platform-agnostic?ref=$PIPELINE_VERSION"

# 3. Espera (10+ min)
kubectl -n kubeflow wait --for=condition=available --timeout=15m deploy --all

# 4. UI
kubectl -n kubeflow port-forward svc/ml-pipeline-ui 8080:80
# http://localhost:8080
```

### Pipeline example (Python SDK)

```python
import kfp
from kfp import dsl

@dsl.component
def add(a: int, b: int) -> int:
    return a + b

@dsl.pipeline(name="demo-add")
def demo(a: int = 2, b: int = 3):
    add(a=a, b=b)

client = kfp.Client(host="http://localhost:8080")
client.create_run_from_pipeline_func(demo, arguments={"a": 10, "b": 20})
```

### Cleanup

```bash
kind delete cluster --name kubeflow-poc
```

### Tips de SRE

- **Instalação completa (Kubeflow all)**: use [charmed-kubeflow](https://charmed-kubeflow.io) ou manifest oficial `manifests/`. 20+GB RAM required.
- **KFP SDK v2** → APIs ficaram diferentes do v1. Pin a versão.
- **Artifacts em MinIO/S3**: modelos, metrics, logs; configurar `PipelineRoot` bucket externo.
- **Kubeflow Profiles** (multi-tenant): cada user/team tem namespace próprio isolado.
- **GPU scheduling**: `nodeSelector` + NVIDIA device plugin. Tolerations para taint `nvidia.com/gpu`.
- Em prod, use **KFP on AWS/GCP/Vertex AI** (managed) — operar Kubeflow full é trabalhoso.

### References

- https://www.kubeflow.org/docs/
- https://www.kubeflow.org/docs/components/pipelines/legacy-v1/installation/localcluster-deployment/
