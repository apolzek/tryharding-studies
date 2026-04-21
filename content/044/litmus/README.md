---
title: Litmus — chaos engineering com marketplace de experimentos
tags: [cncf, incubating, litmus, chaos-engineering, resilience]
status: stable
---

## Litmus (CNCF Incubating)

**O que é:** framework de chaos engineering com **ChaosCenter** (UI central multi-cluster) + **ChaosHub** (marketplace de experimentos prontos) + **CRDs** (`ChaosEngine`, `ChaosExperiment`, `ChaosResult`). Suporta chaos também em aplicações não-k8s (Linux/bare metal).

**Litmus vs Chaos Mesh:**

| | Litmus | Chaos Mesh |
|-|--------|-----------|
| UI | ChaosCenter rica | Chaos Dashboard (mais simples) |
| Hub de experimentos | ChaosHub (git-backed) | YAMLs próprios |
| Workflow | Argo Workflows integrado | Workflow CRD próprio |
| Fora do k8s | Sim (VM, bare-metal) | Só k8s |
| SaaS | Litmus Cloud | Não |

**Quando usar (SRE day-to-day):**

- Quer marketplace de experimentos (pod-delete, node-drain, network-loss, disk-fill, pod-cpu-hog...).
- Multi-cluster: 1 ChaosCenter orquestra experimentos em N clusters.
- Pipeline GitOps de experimentos (CI → rodar experimento A → pass/fail → promote).

### Cenário real

*"Quero uma plataforma central para meus devs auto-servirem experimentos de caos, com histórico, aprovação e catálogo."*

### Reproducing

```bash
cd content/044/litmus

# 1. Cluster
kind create cluster --config kind.yaml

# 2. Install
helm repo add litmuschaos https://litmuschaos.github.io/litmus-helm/
helm install litmus litmuschaos/litmus-3-0-0-beta --version 3.0.0-beta9 \
  --namespace litmus --create-namespace \
  --wait --timeout 5m 2>/dev/null \
  || helm install litmus litmuschaos/litmus --namespace litmus --create-namespace --wait --timeout 5m

kubectl -n litmus get pods
```

### ChaosCenter UI

```bash
kubectl -n litmus port-forward svc/litmusportal-frontend-service 9091:9091
# http://localhost:9091
# user: admin / password: litmus
```

Na UI: criar novo cluster, puxar experimento do ChaosHub (ex: `pod-delete`), rodar contra um deploy vítima.

### Experimento CRD (sem UI)

```bash
# Instala experiment do hub
kubectl apply -f https://hub.litmuschaos.io/api/chaos/3.12.0?file=charts/generic/pod-delete/experiment.yaml -n default
kubectl apply -f https://hub.litmuschaos.io/api/chaos/3.12.0?file=charts/generic/pod-delete/rbac.yaml -n default

# Target
kubectl create deploy victim --image=nginx --replicas=3 -n default
kubectl label deploy victim app=victim -n default

# Engine
cat <<EOF | kubectl apply -f -
apiVersion: litmuschaos.io/v1alpha1
kind: ChaosEngine
metadata:
  name: victim-chaos
  namespace: default
spec:
  appinfo:
    appns: default
    applabel: app=victim
    appkind: deployment
  engineState: active
  chaosServiceAccount: litmus-admin
  experiments:
    - name: pod-delete
      spec:
        components:
          env:
            - name: TOTAL_CHAOS_DURATION
              value: "60"
            - name: PODS_AFFECTED_PERC
              value: "50"
EOF

kubectl get chaosengine,chaosresult -w
```

### Cleanup

```bash
kind delete cluster --name litmus-poc
```

### Tips de SRE

- **SLO gate** (Probe): ChaosEngine pode ter probe HTTP/Prometheus; se o SLO cai durante o experimento, Litmus marca **failed**.
- **BYOChaos**: seu próprio experimento como container. Útil para chaos custom (ex: corromper row do banco).
- **Argo Workflows**: Litmus usa por baixo — orquestra experiments em sequência/paralelo.
- **Cloud**: Litmus também tem experimentos AWS/GCP/Azure (ec2-terminate, etc).
- Nunca rode em prod sem dry-run prévio em stg + janela + observabilidade ativa.

### References

- https://docs.litmuschaos.io/
- https://hub.litmuschaos.io/
