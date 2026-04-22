---
title: Observability-as-a-Service — per-tenant isolated stack on Kubernetes
tags: [observability, otel, kubernetes, helm, go, react, multi-tenant]
status: stable
---

# 048 — Observability-as-a-Service

Self-serve observability platform. User signs up on the front, control plane
provisions an isolated namespace containing a dedicated OTel Collector (with
JWT-gated ingest), VictoriaMetrics, Jaeger, ClickHouse and a pre-configured
Grafana. User gets back a collector endpoint, an ingest token, and a Grafana
URL + credentials. Nothing else is reachable.

## Architecture

```
                ┌──────────────────────── control plane ────────────────────────┐
                │                                                                │
   browser ──► frontend (vite/react) ──► api-gateway (go) ──► auth-service (go) │
                                             │                      │            │
                                             ▼                      ▼            │
                                         provisioner (go)       postgres          │
                                             │  (helm SDK,                        │
                                             │   client-go)                       │
                └──────────────────────────── │ ────────────────────────────────┘
                                              ▼
                                  ┌───────── kind/k8s ─────────┐
                                  │                             │
                                  │  namespace tenant-<id>      │
                                  │  ┌──────────────────────┐   │
                                  │  │ auth-proxy (go)      │   │
                                  │  │  ↓ JWT check         │   │
                                  │  │ otel-collector       │   │
                                  │  │  ├─► victoria-metrics │   │
                                  │  │  ├─► jaeger           │   │
                                  │  │  └─► clickhouse       │   │
                                  │  │ grafana              │   │
                                  │  │   (datasources set)  │   │
                                  │  └──────────────────────┘   │
                                  │  NetworkPolicy, Quota, RBAC  │
                                  └─────────────────────────────┘
```

## Key design decisions

| Decision | Chose | Rejected | Why |
|---|---|---|---|
| Tenant isolation | **namespace** per tenant | vcluster | vcluster × 1000 = 1000 control planes. Namespace + NetworkPolicy + Quota + RBAC covers the threat model at 1/10 the footprint. See `docs/scaling.md`. |
| Provisioning | **Helm** via Go SDK | raw manifests / Kustomize / operator | Single chart, `helm upgrade` is the upgrade story, `helm rollback` is the rollback story. Versions live in one `values.yaml`. |
| Ingest auth | **Go auth-proxy** in front of collector | collector OIDC extension | OTel auth extensions are coarse and per-pipeline. A tiny Go reverse proxy validates JWT with the tenant's claim before forwarding OTLP. |
| Trace backend | **Jaeger (badger) + ClickHouse sink** | Jaeger-on-OpenSearch | User asked no OpenSearch. Jaeger all-in-one with badger for query; collector also writes traces to ClickHouse for long retention. |
| Long-term storage | **ClickHouse** single-node (logs, traces, metrics) | single shared CH | User requested per-tenant. Tradeoff documented in `docs/scaling.md`. |
| Control plane | **Go microservices** | monolith | User asked. Three services sharing Postgres. |
| Frontend | **React + Vite** | Next.js | Static SPA is enough; easier to ship. |

## Repo layout

```
048/
├── README.md                     (this)
├── Makefile                      ( up / test / destroy )
├── docker-compose.yml            ( control-plane local dev: postgres, services )
├── go.work                       ( multi-module workspace )
├── charts/
│   └── tenant-stack/             ( the per-tenant Helm chart )
├── services/
│   ├── auth/                     ( registration, login, JWTs )
│   ├── provisioner/              ( calls Helm to create tenant ns )
│   ├── api-gateway/              ( single entrypoint, routes, auth middleware )
│   ├── collector-proxy/          ( sidecar: JWT check → otel collector )
│   └── shared/                   ( jwt, log, config, db helpers )
├── frontend/                     ( Vite + React, black/white 3D )
├── deploy/
│   ├── kind/kind-config.yaml
│   └── control-plane/            ( k8s manifests for the control plane itself )
├── tests/
│   ├── integration/              ( provision a tenant in kind, send OTLP, query )
│   └── e2e/                      ( sign up → dashboard → ingest )
├── docs/
│   ├── scaling.md                ( 1000-tenant math, shard plan )
│   └── upgrades.md               ( rolling helm upgrade playbook )
└── hack/                         ( scripts )
```

## Running locally

Prereqs: Docker, Go 1.22+, Node 20+, [kind](https://kind.sigs.k8s.io/),
[helm](https://helm.sh/) 3.14+, [kubectl](https://kubernetes.io/docs/tasks/tools/).

```bash
make up          # boots kind, control plane, frontend
make signup      # curl test — registers a tenant, prints credentials
make test        # go test ./... + vitest + integration
make destroy     # nuke kind cluster + compose
```

Frontend is served at `http://localhost:5173`. Control plane API at
`http://localhost:8080`. Per-tenant collector/Grafana are exposed as
`http://tenant-<id>.localtest.me` via ingress (localtest.me resolves to 127.0.0.1).

## Security model

1. Signup → `auth-service` creates a user row, generates a **tenant id** (ULID) and a **long-lived ingest JWT** signed with HS256 (claim `tid`).
2. `provisioner` calls `helm install tenant-stack --namespace tenant-<id>` with values including the JWT signing key as a secret.
3. `collector-proxy` pod validates incoming OTLP requests: `Authorization: Bearer <jwt>`, signature check, `tid == expected`. On fail → 401. On success → forwards to `localhost:4318/4317`.
4. `NetworkPolicy` blocks egress from the tenant ns except to the internet (for the collector's own upstream — optional — disabled by default).
5. Tenant user only ever sees: collector URL + token, Grafana URL + creds. Everything else is cluster-internal.
6. Grafana admin password is generated per tenant, stored hashed in Postgres, shown **once** in the UI at signup time.

## What's production-ready vs stub

| | Status |
|---|---|
| Namespace isolation, NetworkPolicy, Quota | ✅ real |
| Helm-driven provisioning | ✅ real |
| JWT ingest auth | ✅ real |
| Per-tenant OTel → VM/Jaeger/ClickHouse pipeline | ✅ real |
| Grafana pre-configured datasources | ✅ real |
| Rolling upgrade playbook | ✅ documented |
| Postgres HA, Helm retries, leader election | ⚠️ stubbed — see `docs/upgrades.md` |
| Multi-cluster sharding for >1 cluster worth of tenants | ⚠️ architected, not coded — see `docs/scaling.md` |
| TLS on collector ingest | ⚠️ cert-manager hook exists, off by default locally |

## References

- OpenTelemetry Collector config: https://opentelemetry.io/docs/collector/configuration/
- Jaeger + ClickHouse plugin: https://github.com/jaegertracing/jaeger-clickhouse
- VictoriaMetrics sizing: https://docs.victoriametrics.com/Single-server-VictoriaMetrics.html#capacity-planning
- Helm Go SDK: https://pkg.go.dev/helm.sh/helm/v3
