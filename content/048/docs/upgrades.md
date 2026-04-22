# Upgrading component versions across all tenants

All versions live in one place: `charts/tenant-stack/values.yaml`.

```yaml
victoriametrics:
  image: { repository: victoriametrics/victoria-metrics, tag: "v1.106.0" }
jaeger:
  image: { repository: jaegertracing/all-in-one, tag: "1.62.0" }
collector:
  image: { repository: otel/opentelemetry-collector-contrib, tag: "0.115.0" }
clickhouse:
  image: { repository: clickhouse/clickhouse-server, tag: "24.8.4.13-alpine" }
grafana:
  image: { repository: grafana/grafana, tag: "11.3.0" }
```

## The normal rollout

1. Bump the tag in `values.yaml` and bump `Chart.yaml:version` + `appVersion`.
2. Package + publish the chart (optional — for local dev the `provisioner`
   reads the path directly).
3. For every tenant: `helm upgrade tenant charts/tenant-stack -n tenant-<id>`.
4. `helm history` on each namespace gives a rollback point.

## Automated rollout

Add a provision-jobs row per tenant with `kind=upgrade`. The provisioner
already handles this branch (`install-or-upgrade` checks list → picks path).

```sql
-- canary 1%, then batch the rest
INSERT INTO provision_jobs (tenant_id, kind)
SELECT id, 'upgrade' FROM tenants
WHERE status = 'ready'
ORDER BY random()
LIMIT 10;

-- after validation:
INSERT INTO provision_jobs (tenant_id, kind)
SELECT id, 'upgrade' FROM tenants WHERE status = 'ready'
AND id NOT IN (SELECT tenant_id FROM provision_jobs WHERE kind='upgrade' AND finished_at IS NOT NULL);
```

The worker pool polls `provision_jobs` (already FOR UPDATE SKIP LOCKED — safe
to scale horizontally once leader election is added; see scaling.md).

## Rate limiting

A worker-pool of `N` workers with `P` concurrent helm upgrades caps API-server
churn. For 1000 tenants, `N=10, P=5` means ~50 parallel `helm upgrade`s —
well within a 3-master control plane's tolerance, finishes in ~20 minutes if
each upgrade is ~5s.

## Storage-migration upgrades

Some bumps require data migration (ClickHouse major versions, VM storage
format changes).

1. Freeze writes: set the `NetworkPolicy` on the collector to deny, or set
   `otel-collector replicas=0` via a values override.
2. Run a Helm **pre-upgrade hook** that takes a snapshot/backup.
3. Bump the image, run migration.
4. Unfreeze.

The chart doesn't implement this today — it's a template hook (`Job` with
`helm.sh/hook: pre-upgrade`). Ship it when a breaking version arrives.

## Rollback

`helm rollback tenant <revision> -n tenant-<id>`

The provisioner exposes this as job `kind=rollback` (not yet wired — one-liner
addition). For emergencies operators can `kubectl` direct.

## Keeping versions consistent across tenants

Track the deployed chart version in the `tenants.chart_version` column
(already written by the provisioner on successful install/upgrade). A simple
query gives drift:

```sql
SELECT chart_version, count(*) FROM tenants WHERE status='ready' GROUP BY 1;
```

If a tenant is stuck on an old chart_version, enqueue an `upgrade` job.

## Why Helm over an Operator

Operators (Kubebuilder/operator-sdk) are better when you need ongoing
reconciliation (watching CRs and reacting). For observability stacks the
state is "install these 6 deployments with these values" — there's nothing
to reconcile that `helm upgrade` doesn't already handle. The operator adds
a CRD, a controller, and a webhook — all overhead we don't need at this
stage. If per-tenant CR editing ever becomes a product requirement
(e.g. "a tenant should be able to ask for VM retention change via API"),
that's when to move to an operator.
