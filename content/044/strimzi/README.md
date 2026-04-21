---
title: Strimzi — Kafka operator para Kubernetes
tags: [cncf, incubating, strimzi, kafka, operator, streaming]
status: stable
---

## Strimzi (CNCF Incubating)

**O que é:** Kubernetes operator que gerencia clusters Apache Kafka. Você escreve `Kafka`, `KafkaTopic`, `KafkaUser`, `KafkaConnect`, `KafkaMirrorMaker2` como YAML e o operator provisiona + rebalance + upgrade.

**Quando usar (SRE day-to-day):**

- Self-hosted Kafka em Kubernetes (on-prem ou cloud).
- Multi-cluster stretch (MirrorMaker2).
- Operador maduro - faz rolling upgrade, JBOD, storage expansion, Cruise Control auto-rebalance.
- Alternativa a AWS MSK / Confluent Cloud se custo/controle justifica.

**Quando NÃO usar:**

- Equipe sem expertise Kafka — managed (MSK/Confluent) evita dor.
- Workload simples pub/sub — NATS JetStream é mais leve.

### Cenário real

*"Precisamos de um Kafka cluster para event streaming, on-prem. Queremos operar declarativamente + topic-as-code."*

### Reproducing

```bash
cd content/044/strimzi

# 1. Cluster
kind create cluster --config kind.yaml

# 2. Operator
kubectl create namespace kafka
kubectl apply -f 'https://strimzi.io/install/latest?namespace=kafka' -n kafka
kubectl -n kafka wait --for=condition=available --timeout=5m deploy/strimzi-cluster-operator

# 3. Kafka cluster + topic
kubectl apply -f kafka.yaml -n kafka

# 4. Espera Kafka ficar Ready (~3 min)
kubectl -n kafka wait kafka/demo --for=condition=Ready --timeout=10m
kubectl -n kafka get kafka,kafkatopics
```

### Produzir / consumir

```bash
# Producer
kubectl -n kafka run kafka-producer -ti --image=quay.io/strimzi/kafka:0.44.0-kafka-3.8.0 --rm=true --restart=Never -- \
  bin/kafka-console-producer.sh --bootstrap-server demo-kafka-bootstrap:9092 --topic orders
# digite mensagens, ENTER

# Consumer (outro terminal)
kubectl -n kafka run kafka-consumer -ti --image=quay.io/strimzi/kafka:0.44.0-kafka-3.8.0 --rm=true --restart=Never -- \
  bin/kafka-console-consumer.sh --bootstrap-server demo-kafka-bootstrap:9092 --topic orders --from-beginning
```

### Cleanup

```bash
kind delete cluster --name strimzi-poc
```

### Tips de SRE

- **KRaft** (sem Zookeeper) é o default moderno - use. Menos processos para operar.
- **Cruise Control** (Strimzi integra): auto-rebalance + detecta hot partitions.
- **Storage**: JBOD com múltiplos PVCs por broker — mais IO, mais resiliência.
- **User Operator**: `KafkaUser` cria ACL + scram credentials declarativamente.
- **Upgrade**: operator faz minor Kafka upgrade sozinho. Major (3.x → 4.x) tem procedimento (metadataVersion).
- Em prod: replication.factor=3, min.insync.replicas=2, acks=all.

### References

- https://strimzi.io/docs/
- https://strimzi.io/quickstarts/
