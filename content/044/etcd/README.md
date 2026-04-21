---
title: etcd — cluster KV distribuído com quorum Raft
tags: [cncf, graduated, etcd, kv, raft, distributed]
status: stable
---

## etcd (CNCF Graduated)

**O que é:** key-value store distribuído e strongly consistent via **Raft**. Core do Kubernetes (`kube-apiserver` escreve tudo em etcd). Também usado standalone: service discovery, leader election, locks distribuídos, feature flags.

**Quando usar (SRE day-to-day):**

- Backing store do Kubernetes (já vem — o job SRE é operar, fazer backup, restore).
- Leader election em apps HA — "quem é o active?" → etcd lease + lock.
- Config distribuída com watch (fan-out de "feature X mudou" p/ N réplicas).
- Coordenação fraca mas forte (locks com TTL).

**Quando NÃO usar:**

- Dataset grande — etcd guarda tudo em memória; alvo é <2GB. Não é banco de dados.
- Escritas massivas — Raft é sequential commit. Consul/Cassandra escalam horizontal melhor.

### Cenário real

*"Quero um cluster etcd 3-node para treinar backup/restore do kube-apiserver, simular perda de um nó e ver se continua aceitando writes."*

### Reproducing

```bash
cd content/044/etcd
docker compose up -d
sleep 5
```

Exercite o cluster com `etcdctl`:

```bash
# Escreve via node 1, lê via qualquer — linearizable read
docker exec cncf-etcd1 etcdctl put /app/config/featureA "on"
docker exec cncf-etcd2 etcdctl get /app/config/featureA
docker exec cncf-etcd3 etcdctl get /app/config/featureA --print-value-only

# Health do cluster
docker exec cncf-etcd1 etcdctl --endpoints=etcd1:2379,etcd2:2379,etcd3:2379 \
  endpoint status --write-out=table

# Member list
docker exec cncf-etcd1 etcdctl member list --write-out=table
```

### Simulando falha de 1 nó

```bash
# Para o leader
docker stop cncf-etcd1

# Cluster continua (tolerância = (N-1)/2 = 1 com 3 nós)
docker exec cncf-etcd2 etcdctl put /test/after-failure "ok"
docker exec cncf-etcd3 etcdctl get /test/after-failure

# Volta
docker start cncf-etcd1
```

### Backup / restore (skill essencial de SRE k8s)

```bash
docker exec cncf-etcd1 etcdctl snapshot save /tmp/snap.db
docker exec cncf-etcd1 etcdctl snapshot status /tmp/snap.db --write-out=table
# Em produção esse snapshot vai para S3 via cron.
```

### Cleanup

```bash
docker compose down -v
```

### Tips de SRE

- **Número ímpar de nós** (3/5/7). 3 = quorum 2, tolera 1; 5 = tolera 2.
- **Disco rápido**: etcd é sensível a fsync latency. SSD NVMe em produção — HDD mata o cluster.
- **Snapshot periódico** (backup): `etcdctl snapshot save` + upload p/ S3 a cada 30min. Em k8s hosted (GKE/EKS), isso é gerenciado.
- **Alertas**: `etcd_server_leader_changes_seen_total` — flap = rede ruim entre masters. `etcd_mvcc_db_total_size_in_bytes` > 2GB = hora de compact + defrag.
- **`etcdctl defrag`** após muitos deletes — senão o DB não encolhe.

### References

- https://etcd.io/docs/
- https://kubernetes.io/docs/tasks/administer-cluster/configure-upgrade-etcd/
