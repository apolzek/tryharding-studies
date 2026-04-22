# Scaling to 1000 tenants

## Per-tenant resource budget

With the conservative `values.yaml` defaults (`resources.small`/`medium`):

| Component        | CPU req | Mem req | Mem limit | PVC   |
|------------------|--------:|--------:|----------:|------:|
| otel-collector   |   50m   |  64Mi   |  512Mi    | —     |
| auth-proxy       |   50m   |  64Mi   |  256Mi    | —     |
| victoriametrics  |   50m   |  64Mi   |  512Mi    |  5Gi  |
| jaeger           |   50m   |  64Mi   |  512Mi    |  2Gi  |
| clickhouse       |  100m   | 128Mi   |  512Mi    | 10Gi  |
| grafana          |   50m   |  64Mi   |  512Mi    |  1Gi  |
| **sum**          | **350m**| **448Mi** | **2.8Gi** | **18Gi** |

A namespace `ResourceQuota` caps the hard ceiling at 4 vCPU / 6 GiB / 20 GiB.

## 1000-tenant math

| Resource | Per-tenant (ceiling) | Target × 1000 |
|---|---:|---:|
| CPU request (aggregate) | 350m        | **350 vCPU** |
| Memory request          | 0.45 GiB    | **~450 GiB** |
| PVC                     | 18 GiB      | **~18 TiB** |
| Pods                    | ~8          | **~8000** |

Conclusion: **one Kind cluster cannot hold 1000 tenants.** Plans:

### Plan A — shard across clusters (recommended)

The control plane already speaks to a kubeconfig; generalize the `provisioner`
to pick a target cluster by hashing `tenant_id`. Each cluster sized for
~100–150 tenants (≈50 vCPU, 75 GiB RAM, ~3 TiB storage, fits on a 3-node
`c5.9xlarge`-class group + EBS). Adding capacity = add a cluster, register it
in the `clusters` table.

Implementation delta (small):

1. `clusters(id, name, kubeconfig, region, capacity, current)` table.
2. Provisioner reads target cluster from `tenants.cluster_id`, built from
   `crc32(tenant_id) % len(active_clusters)` at registration.
3. DNS: a wildcard `*.obs.example.com` points at a shared ingress that
   routes by `host` header — either via a GSLB or one ingress per cluster.

### Plan B — share ClickHouse across tenants (cost-cut)

ClickHouse at rest is the biggest line item (10 GiB × 1000 = 10 TiB). A
shared CH cluster with per-tenant database gives ~10× compression gains and
~5× lower RAM, but means CH is now a shared blast-radius. Keep it optional /
per-tier (“Enterprise” = dedicated, “Pro” = shared).

### Plan C — cold-path to object storage (VM + Jaeger + CH)

VictoriaMetrics has `vmstorage` + object-storage retention; ClickHouse has
disk tiering to S3. For data-retention past 30 days, tier to S3 and back the
“hot” volume on SSD. Reduces PVC pressure by ~90% for the long tail.

## What the chart gives you today

- `ResourceQuota` + `LimitRange`: no tenant can blow past 4 vCPU / 6 GiB.
- `NetworkPolicy`: default-deny, then allow intra-ns + DNS + ingress-nginx.
- Separate `StatefulSet`s for stateful components so PVCs survive pod rolls.

## What is NOT in the POC but matters at 1000 tenants

- **Leader election** for the provisioner. Right now a single pod polls. Wrap
  the tick loop in `leaderelection.RunOrDie` from `client-go`. Multiple
  replicas become safe.
- **Bulk operations API**. `helm upgrade` per tenant doesn't scale beyond a
  few hundred because release churn hits tiller (now: helm 3 secrets in each
  namespace). Batch upgrades with a worker pool + rate limit; see
  `docs/upgrades.md`.
- **Cert management**. For real tenants, TLS at the ingress is non-negotiable.
  `cert-manager` + `HTTP01` solver per ingress; or one wildcard cert per
  cluster managed via DNS01. Hooks for this live in the ingress annotations
  but are off by default.
- **Cost observability**. Mix in `opencost` to attribute spend back to
  `obs-saas.io/tenant` label.
