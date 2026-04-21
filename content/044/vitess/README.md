---
title: Vitess — sharding horizontal de MySQL em escala YouTube
tags: [cncf, graduated, vitess, mysql, sharding, database]
status: stable
---

## Vitess (CNCF Graduated)

**O que é:** sistema de clustering+sharding para MySQL criado no YouTube. Coloca um proxy SQL (`vtgate`) entre o cliente e os shards, faz routing por vindex, resharding online, failover, backup. Seu app fala MySQL wire protocol; Vitess distribui em N instâncias reais.

**Quando usar (SRE day-to-day):**

- MySQL que cresceu tanto que 1 instância não aguenta (dataset > TB, QPS > 100k).
- Multi-tenant SaaS onde cada cliente vai para um shard diferente (tenant-based routing).
- Precisa de resharding ZERO-DOWNTIME (Vitess faz online resharding — o diferencial).

**Quando NÃO usar:**

- Dataset pequeno — complexidade brutal. 1 RDS Multi-AZ resolve.
- Joins cross-shard complexos — Vitess tenta, mas é melhor evitar.
- Postgres — Vitess é MySQL only (para Postgres: Citus, CockroachDB, YugabyteDB).

### Cenário real

*"Meu MySQL de 2TB virou gargalo. Quero sharding por `customer_id` mas não posso ter downtime."*

### Reproducing

```bash
cd content/044/vitess

# 1. Cluster
kind create cluster --config kind.yaml

# 2. Install operator
kubectl apply -f https://raw.githubusercontent.com/vitessio/vitess/main/examples/operator/operator.yaml

# 3. Config (init SQL + users p/ vtgate)
kubectl create secret generic example-cluster-config \
  --from-literal=users.json='{"user":[{"UserData":"user","Password":""}]}' \
  --from-literal=init_db.sql='CREATE TABLE IF NOT EXISTS demo (id INT PRIMARY KEY);'

# 4. VitessCluster
kubectl apply -f cluster.yaml

# 5. Observa (pods sobem em ordem: etcd-global → vtctld → vttablet → vtgate)
kubectl get pods -w
kubectl get vitessclusters,vitessshards
```

### Conectando

```bash
# porta do vtgate (MySQL wire protocol)
kubectl port-forward svc/example-zone1-vtgate-bc6cde92 15306:15306

# em outro terminal
mysql -h 127.0.0.1 -P 15306 -u user
# e você fala MySQL normal — Vitess distribui.
```

### Cleanup

```bash
kind delete cluster --name vitess-poc
```

### Tips de SRE

- **VSchema** define vindexes (como shardear). Errar o vindex = hot shard. Teste com amostras de produção antes.
- **Online resharding**: `vtctldclient Reshard Create` divide/funde shards com replicação stream. Zero downtime real.
- **PlannedReparentShard** para failover controlado. Não mate o primary na mão.
- **Backup**: xtrabackup para S3 via `BackupLocation`. Cron `BackupSchedule` — em prod, 1x/dia por shard.
- **vtadmin** é a UI (web) — útil p/ debug em prod.
- Em prod não use kind — use ao menos 3 nós worker e PVCs em storage rápido.

### References

- https://vitess.io/docs/
- https://github.com/planetscale/vitess-operator
