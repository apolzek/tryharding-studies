---
title: Longhorn — block storage distribuído cloud-native (Rancher)
tags: [cncf, incubating, longhorn, storage, block, csi]
status: stable
---

## Longhorn (CNCF Incubating)

**O que é:** block storage distribuído 100% em user space (sem kernel mod) para Kubernetes. Cria replicas sincronas em múltiplos nós + UI web + snapshots + backup para S3/NFS. Mais leve operacionalmente que Rook-Ceph.

**Longhorn vs Rook-Ceph:**

| | Longhorn | Rook-Ceph |
|-|----------|-----------|
| Protocol | Block (RBD-like) | Block + File + Object |
| Setup | 1 chart, pronto | Operator + Cluster + Pools |
| Operação | Muito simples | Requer expertise Ceph |
| Escala | ~100 nós | 1000+ nós comprovado |
| Features | Snapshots, backup, DR | Tudo + CephFS + S3 |

Para 90% dos clusters que só precisam de PVC block resiliente, Longhorn é a escolha simples. Rook quando precisa de object/file ou escala massiva.

**Quando usar (SRE day-to-day):**

- On-prem / cluster sem cloud storage provider.
- Precisa de backup automático de PV para S3 externo (DR).
- Snapshots para rollback de PVC (test data, upgrade de database).
- Homelab / edge — requer só Linux.

### Cenário real

*"On-prem k3s/kind. Quero PV dinâmico com 2 replicas em nós diferentes + snapshot diário para S3. Não quero operar Ceph."*

### Reproducing

```bash
cd content/044/longhorn

# 1. Cluster (3 nós para permitir 2 replicas)
kind create cluster --config kind.yaml

# Longhorn precisa de iscsiadm + open-iscsi no host. Docker Desktop/kind vanilla
# não tem. Se rodar kind direto no Linux host:
sudo apt-get install -y open-iscsi nfs-common
sudo systemctl enable --now iscsid

# 2. Install Longhorn
kubectl apply -f https://raw.githubusercontent.com/longhorn/longhorn/v1.7.2/deploy/longhorn.yaml
kubectl -n longhorn-system wait --for=condition=available --timeout=8m deploy --all

# 3. Cria PVC + app que escreve
kubectl apply -f pvc-app.yaml
kubectl get pvc,sts demo-writer -w
```

### UI (útil para troubleshoot)

```bash
kubectl -n longhorn-system port-forward svc/longhorn-frontend 8080:80
# http://localhost:8080
```

### Snapshot + backup

```bash
# Cria snapshot (API via kubectl exec no manager, ou via UI)
kubectl -n longhorn-system apply -f - <<EOF
apiVersion: longhorn.io/v1beta2
kind: Snapshot
metadata:
  name: demo-snap-1
  namespace: longhorn-system
spec:
  volume: pvc-<uuid>   # pegue do kubectl get pv
EOF
```

Para backup S3, configure `BackupTarget` na UI/CRD apontando para o bucket.

### Cleanup

```bash
kind delete cluster --name longhorn-poc
```

### Tips de SRE

- **open-iscsi** precisa estar no host — sem isso, Longhorn não anexa volumes.
- **Replica count**: default 3. Diminua para 2 em cluster de 2 nós, nunca 1 em prod.
- **Backup target S3**: configure desde o dia 1. DR = "perdi o cluster inteiro → restauro do S3 em novo cluster".
- **Data locality**: `data-locality: best-effort` roda o pod no mesmo nó da replica → IO local.
- **Defrag de volume** `RecurringJob` tipo `filesystem-trim` para thin-provisioning recuperar espaço.
- Upgrade chart é sensível — leia release notes e teste em cluster separado.

### References

- https://longhorn.io/docs/
- https://longhorn.io/docs/1.7.2/best-practices/
