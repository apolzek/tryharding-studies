---
title: Chaos Mesh — engenharia de caos declarativa em k8s
tags: [cncf, incubating, chaos-mesh, chaos-engineering, resilience]
status: stable
---

## Chaos Mesh (CNCF Incubating)

**O que é:** plataforma de chaos engineering. Injeta falhas controladas no cluster via CRDs: **PodChaos** (kill/pause), **NetworkChaos** (latência, perda, partição), **StressChaos** (CPU/mem burn), **IOChaos** (delay/error em filesystem), **TimeChaos** (pular relógio), **HTTPChaos** (latência em L7), **DNSChaos**, **KernelChaos**, etc.

**Quando usar (SRE day-to-day):**

- GameDay — treina o time para falhas que vão acontecer em prod de madrugada.
- Validar hipóteses: "meu serviço aguenta perder 1 pod aleatório?" → PodChaos + assert SLO.
- CI/CD: rodar ChaosExperiment no stage, se errar SLO → falha deploy.
- Simular AZ down (NetworkChaos `partition` entre subsets de nós).

**Quando NÃO usar:**

- Prod sem autorização e sem blast radius controlado — você vai se machucar.
- Time sem observabilidade (SLO + alertas) — chaos sem sinais = só quebrar por quebrar.

### Cenário real

*"Meu deployment tem 3 réplicas. Quero validar que, se 1 pod for morto a cada minuto, o serviço continua aceitando requests com 0 erros."*

### Reproducing

```bash
cd content/044/chaos-mesh

# 1. Cluster
kind create cluster --config kind.yaml

# 2. Install
helm repo add chaos-mesh https://charts.chaos-mesh.org
helm install chaos-mesh chaos-mesh/chaos-mesh \
  --version 2.7.0 \
  -n chaos-mesh --create-namespace \
  --set chaosDaemon.runtime=containerd \
  --set chaosDaemon.socketPath=/run/containerd/containerd.sock \
  --wait --timeout 5m

# 3. Workload vítima
kubectl create deploy victim --image=nginx --replicas=3
kubectl label deploy victim app=victim --overwrite

# 4. Experimento
kubectl apply -f experiment.yaml

# 5. Observa: chaos mata 1 pod a cada minuto. ReplicaSet recria.
kubectl get pods -l app=victim -w
```

### Dashboard (UI de experimentos)

```bash
kubectl -n chaos-mesh port-forward svc/chaos-dashboard 2333:2333
# http://localhost:2333
```

### Cleanup

```bash
kind delete cluster --name chaos-mesh-poc
```

### Exemplos úteis

```yaml
# NetworkChaos: adiciona 500ms de latência entre frontend e backend
kind: NetworkChaos
spec:
  action: delay
  mode: all
  selector:
    labelSelectors: { app: backend }
  delay:
    latency: "500ms"
    jitter: "100ms"
  duration: "10m"
  direction: to
  target:
    mode: all
    selector:
      labelSelectors: { app: frontend }
```

```yaml
# StressChaos: 80% CPU em 1 pod por 5min
kind: StressChaos
spec:
  mode: one
  selector: { labelSelectors: { app: api } }
  stressors:
    cpu: { workers: 1, load: 80 }
  duration: "5m"
```

### Tips de SRE

- **Workflow CRD**: encadeia experimentos (primeiro NetworkChaos, depois PodChaos, etc) para cenários complexos.
- **Schedule CRD**: roda experimento em cron (GameDay toda sexta às 14h).
- **Blast radius**: `mode: fixed-percent: "10"` afeta 10% dos pods, não 100%.
- **Pause**: anotação `chaos-mesh.org/pause: true` pausa experimento vivo — útil em incidente real.
- **Observabilidade**: alertas "Chaos experiment ativo" no Grafana evitam pânico do oncall.

### References

- https://chaos-mesh.org/docs/
- https://chaos-mesh.org/docs/simulate-pod-chaos-on-kubernetes/
