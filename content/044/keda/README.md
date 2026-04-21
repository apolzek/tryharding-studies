---
title: KEDA — autoscaling dirigido por evento (fila/cron/métrica externa)
tags: [cncf, graduated, keda, autoscaling, hpa, event-driven]
status: stable
---

## KEDA (CNCF Graduated)

**O que é:** Kubernetes Event-Driven Autoscaling. Extende o HPA para escalar baseado em **sinais externos** (fila Kafka com N msgs, latência Prometheus, cron, Azure Service Bus, AWS SQS, Redis, RabbitMQ, etc — 60+ scalers). Inclui **scale-to-zero**, coisa que HPA nativo não faz.

**Quando usar (SRE day-to-day):**

- Worker que lê fila — quando vazia, 0 pod. Quando bate 1000 msgs, sobe para 10.
- Job que só roda de dia (cron scaler) — 0 à noite, 5 em horário comercial.
- Scaling por SLO — latência P95 > 500ms no Prometheus → sobe. Abaixa → desce.
- Cut de custo em dev/stg — tudo dormindo fora do horário útil.

**Quando NÃO usar:**

- Carga 100% CPU-bound estável — HPA nativo (com `cpu:` metric) já resolve.
- Cargas que não toleram cold start — scale-to-zero mata a UX do primeiro request.

### Cenário real

*"Tenho um worker que consome fila Kafka. Fora do horário de pico, a fila fica vazia mas os pods continuam queimando CPU. Quero 0 replica quando não tem msg."*

Este POC usa um **cron scaler** (mais fácil de validar localmente do que fila real): a cada 2min, escala de 0 → 3. Substitui por Kafka/SQS/Prometheus em prod.

### Reproducing

```bash
cd content/044/keda

# 1. Cluster
kind create cluster --config kind.yaml

# 2. Install KEDA via Helm
helm repo add kedacore https://kedacore.github.io/charts
helm repo update
helm install keda kedacore/keda --version 2.16.0 \
  --namespace keda --create-namespace --wait

# 3. Workload + ScaledObject
kubectl apply -f workload.yaml

# 4. Observa scaling
kubectl get hpa,scaledobject
kubectl get deploy worker -w
# aguarde o próximo múltiplo de 2 min: replicas 0 → 3 → 0 → 3 ...
```

### Exemplos de triggers em produção

```yaml
# Kafka: escala até 10 pods se lag > 100 mensagens
triggers:
  - type: kafka
    metadata:
      bootstrapServers: kafka:9092
      topic: orders
      consumerGroup: worker
      lagThreshold: "100"

# Prometheus: escala por taxa de requests
triggers:
  - type: prometheus
    metadata:
      serverAddress: http://prometheus:9090
      metricName: http_requests_total
      threshold: "100"
      query: sum(rate(http_requests_total{service="api"}[1m]))

# AWS SQS
triggers:
  - type: aws-sqs-queue
    metadata:
      queueURL: https://sqs.us-east-1.amazonaws.com/.../my-queue
      queueLength: "50"
      awsRegion: us-east-1
    authenticationRef:
      name: keda-aws-credentials
```

### Cleanup

```bash
kind delete cluster --name keda-poc
```

### Tips de SRE

- **Scale-to-zero**: `minReplicaCount: 0` (o diferencial). Cuidado com cold start — o primeiro evento espera o pod subir.
- **pollingInterval** (default 30s): intervalo com que o KEDA avalia o trigger. Abaixe para reagir mais rápido; cuidado com custo de scrape.
- **cooldownPeriod** (default 300s): evita flap (subiu, baixou, subiu). Aumente se a carga é burst-y.
- **Multiple triggers**: KEDA usa o maior `desired`. Combine cron (horário) + queue (carga real).
- **Autenticação** (Kafka SASL, AWS IRSA, etc): use `TriggerAuthentication` — nunca inline secret.

### References

- https://keda.sh/docs/latest/
- https://keda.sh/docs/latest/scalers/ (lista de todos os scalers)
