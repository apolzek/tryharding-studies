---
title: OpenCost — cost allocation para Kubernetes
tags: [cncf, incubating, opencost, finops, cost, showback]
status: stable
---

## OpenCost (CNCF Incubating)

**O que é:** motor de cost allocation para Kubernetes. Lê métricas do Prometheus (CPU/mem/disk/network requests e usage), cruza com custo de nó (cloud pricing API ou tabela customizada), e te diz **quanto cada namespace / deployment / label custa**.

**Quando usar (SRE day-to-day):**

- FinOps — reportar custo por time/produto (chargeback / showback).
- Identificar "over-provisioning" — pods com `request: 2CPU` usando 0.1 = dinheiro jogado fora.
- Comparar custo entre clusters/regiões/ambientes.
- Budget alerts: "prod:ns-payments gastou > $5k este mês".

**Quando NÃO usar:**

- Cloud com billing detalhado já (AWS Cost Explorer + tags + daily export) pode ser suficiente se workload é estável.
- Cluster muito pequeno — overhead do OpenCost/Prometheus maior que a economia.

### Cenário real

*"Diretor financeiro quer saber quanto cada time gasta em k8s este mês. Cloud bill agrega por conta, não por namespace."*

### Reproducing

```bash
cd content/044/opencost

# 1. Cluster
kind create cluster --config kind.yaml

# 2. Instala Prometheus (depend)
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm install prometheus prometheus-community/prometheus --version 25.27.0 \
  --namespace prometheus --create-namespace \
  --set extraScrapeConfigs='- job_name: opencost
    static_configs:
    - targets: ["opencost.opencost:9003"]' \
  --wait

# 3. Instala OpenCost
helm repo add opencost https://opencost.github.io/opencost-helm-chart
helm install opencost opencost/opencost --version 2.5.14 \
  --namespace opencost --create-namespace \
  --set opencost.prometheus.internal.enabled=true \
  --set opencost.prometheus.internal.serviceName=prometheus-server \
  --set opencost.prometheus.internal.namespaceName=prometheus \
  --set opencost.prometheus.internal.port=80 \
  --wait --timeout 5m

# 4. UI
kubectl -n opencost port-forward svc/opencost 9090:9090
# http://localhost:9090 → painel de custo por namespace
```

### API direta (útil p/ integrar em Grafana/Slack bot)

```bash
kubectl -n opencost port-forward svc/opencost 9003:9003 &
sleep 2

# custo agregado por namespace na última hora
curl -s 'http://localhost:9003/allocation?window=1h&aggregate=namespace' \
  | python3 -m json.tool | head -30
```

### Cleanup

```bash
kind delete cluster --name opencost-poc
```

### Tips de SRE

- **Pricing customizado**: se on-prem / bare-metal, configure custo por CPU/mem/GB disk manualmente — OpenCost não tem API.
- **Network cost**: OpenCost vê tráfego inter-AZ se Prometheus tiver `node_network_transmit_bytes_total`. Configure cloud pricing por GB.
- **Kubecost** (versão paga, same codebase) tem UI mais rica, alerts, idle cost. Comece com OpenCost e migre se precisar.
- **Idle cost**: pod reserva mas não usa → OpenCost reporta `idle`. Primeiro lugar para cortar.
- **Showback > chargeback**: mostrar custo (showback) é menos atrito do que cobrar (chargeback). Comece pelo showback.

### References

- https://www.opencost.io/docs/
- https://github.com/opencost/opencost
