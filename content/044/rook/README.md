---
title: Rook — Ceph operator para storage distribuído em k8s
tags: [cncf, graduated, rook, ceph, storage, block, object, file]
status: stable
---

## Rook (CNCF Graduated)

**O que é:** operator Kubernetes que gerencia ciclo de vida do **Ceph** (storage distribuído). Ceph oferece 3 primitivas: RBD (block), CephFS (file), RGW (S3-compatible object). Rook traduz isso em CRDs (`CephCluster`, `CephBlockPool`, `CephFilesystem`, `CephObjectStore`).

**Quando usar (SRE day-to-day):**

- On-prem/bare-metal onde não há AWS EBS/GCP PD — precisa de PV dinâmico.
- Multi-protocol (app pede block, data pipeline pede S3, usuário pede NFS) — 1 cluster Ceph atende os 3.
- DR multi-região — Ceph suporta replicação assíncrona (rbd-mirror).

**Quando NÃO usar:**

- Cloud managed — use EBS/GP3, GCE PD, Azure Disk. Rook-Ceph tem custo operacional real (monitorar OSDs, balance, resharding).
- Cluster com <3 nós worker — Ceph com replica 3 precisa de 3 nós reais para tolerância.

### Cenário real

*"On-prem data center. Preciso de PVC dinâmico para prod workload, S3 interno para backup, e NFS compartilhado. Budget = hardware que já tenho."*

### Reproducing

> ⚠️ **Limitação do kind**: Rook-Ceph espera **discos reais** (block devices raw) para OSDs. Em kind, isso requer anexar loop devices ou configurar `useAllDevices: false` + `directories` (modo deprecated). Em produção, use nós bare-metal ou cloud com discos dedicados.

```bash
cd content/044/rook

# 1. Cluster (3 workers para tolerância + extraMounts de /dev)
kind create cluster --config kind.yaml

# 2. Em cada worker, crie um loopback device
for i in 0 1 2; do
  sudo dd if=/dev/zero of=/tmp/rook-disk-$i.img bs=1M count=20480
  sudo losetup /dev/loop${i} /tmp/rook-disk-$i.img
done
# Monte no container do kind worker:
# docker exec rook-poc-worker mknod /dev/loop0 b 7 0  (e chmod 660)

# 3. Operator + CephCluster
kubectl apply -f https://raw.githubusercontent.com/rook/rook/v1.15.4/deploy/examples/crds.yaml
kubectl apply -f https://raw.githubusercontent.com/rook/rook/v1.15.4/deploy/examples/common.yaml
kubectl apply -f https://raw.githubusercontent.com/rook/rook/v1.15.4/deploy/examples/operator.yaml
kubectl apply -f https://raw.githubusercontent.com/rook/rook/v1.15.4/deploy/examples/cluster-test.yaml  # test cluster (1 mon, 1 mgr, 1 osd)

kubectl -n rook-ceph get cephcluster -w
```

### Exercitando PVC após Ceph HEALTH_OK

```bash
# StorageClass RBD
kubectl apply -f https://raw.githubusercontent.com/rook/rook/v1.15.4/deploy/examples/csi/rbd/storageclass-test.yaml

# PVC
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: PersistentVolumeClaim
metadata: { name: demo-pvc }
spec:
  accessModes: [ReadWriteOnce]
  resources: { requests: { storage: 1Gi } }
  storageClassName: rook-ceph-block
EOF

kubectl get pvc demo-pvc -w
```

### Toolbox (diagnóstico real de Ceph)

```bash
kubectl apply -f https://raw.githubusercontent.com/rook/rook/v1.15.4/deploy/examples/toolbox.yaml
kubectl -n rook-ceph exec -it deploy/rook-ceph-tools -- ceph status
kubectl -n rook-ceph exec -it deploy/rook-ceph-tools -- ceph osd tree
```

### Cleanup

```bash
kind delete cluster --name rook-poc
sudo losetup -d /dev/loop{0,1,2}
sudo rm /tmp/rook-disk-*.img
```

### Tips de SRE

- **HEALTH_WARN/ERR**: fique de olho em `ceph status` — PGs inactive/undersized = dados em risco.
- **OSD failure domain**: separe por rack/nó/AZ via CRUSH rules. Replica 3 em 3 nós diferentes.
- **Capacity planning**: não encha >80% — Ceph começa a ficar lento; >95% bloqueia escrita.
- **Backup externo**: Rook replica no cluster, mas se perder o cluster inteiro (comando `kubectl delete cephcluster` errado), perdeu tudo. Snapshot RBD → S3 offsite.
- **Upgrade**: sempre minor-version-by-minor (v1.14 → v1.15 → v1.16), nunca pulando.

### References

- https://rook.io/docs/rook/latest-release/
- https://github.com/rook/rook/tree/master/deploy/examples
