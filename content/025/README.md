# PoC 025 — Apache Flink Kubernetes Operator on Kind

This lab spins up a local Kubernetes cluster with [kind](https://kind.sigs.k8s.io/),
installs the [Apache Flink Kubernetes Operator](https://nightlies.apache.org/flink/flink-kubernetes-operator-docs-stable/)
via Helm, and runs a simple streaming job (`StateMachineExample`) as a `FlinkDeployment`
custom resource to validate the end-to-end flow.

## Versions used

| Component                       | Version  | Notes |
|--------------------------------|----------|-------|
| kind                            | `v0.31.0`| Cluster provisioning tool |
| Kubernetes (requested)          | `v1.35.3`| **See note below** |
| Kubernetes (actually deployed)  | `v1.35.1`| Closest `kindest/node` image available on Docker Hub at the time of writing |
| Helm                            | `v4.1.4` | Chart manager |
| cert-manager                    | `v1.20.2`| Required for operator webhooks |
| Flink Kubernetes Operator       | `1.14.0` | Latest stable release (not `latest` tag) |
| Apache Flink runtime image      | `flink:1.20` | Used by the sample `FlinkDeployment` |

> **About Kubernetes v1.35.3**
> The task requested Kubernetes `v1.35.3`. At the time this PoC was created,
> the `kindest/node` image registry only published images up to `v1.35.1`.
> Because kind requires a pre-built node image to boot, the lab uses `v1.35.1`.
> When the `v1.35.3` image lands on Docker Hub, the only change required is
> bumping the `image:` field in `kind-config.yaml`.

## Repository layout

```
025/
├── README.md               # This file
├── kind-config.yaml        # Kind cluster topology (1 cp + 2 workers)
├── values.yaml             # Helm values for the Flink operator
└── flink-example-job.yaml  # Sample FlinkDeployment (StateMachineExample)
```

---

## Step-by-step walkthrough

### 1. Create the kind cluster

`kind-config.yaml` declares a three-node cluster (1 control plane + 2 workers)
pinned to `kindest/node:v1.35.1`.

```bash
kind create cluster --config kind-config.yaml
kubectl get nodes -o wide
```

Expected output (abbreviated):

```
NAME                      STATUS   ROLES           VERSION
flink-lab-control-plane   Ready    control-plane   v1.35.1
flink-lab-worker          Ready    <none>          v1.35.1
flink-lab-worker2         Ready    <none>          v1.35.1
```

### 2. Install cert-manager (operator webhook dependency)

The Flink Operator ships a validating/mutating admission webhook. Its TLS
material is managed by cert-manager, so cert-manager must be installed
**before** the operator.

```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.20.2/cert-manager.yaml
kubectl -n cert-manager wait --for=condition=Available deploy --all --timeout=300s
```

### 3. Install the Flink Kubernetes Operator via Helm

We use the upstream Apache release repository. The chart is served
per-version under `downloads.apache.org/flink/flink-kubernetes-operator-<version>/`.

```bash
helm repo add flink-operator-repo https://downloads.apache.org/flink/flink-kubernetes-operator-1.14.0/
helm repo update

kubectl create namespace flink-operator
kubectl create namespace flink-jobs

helm install flink-kubernetes-operator \
  flink-operator-repo/flink-kubernetes-operator \
  --version 1.14.0 \
  --namespace flink-operator \
  -f values.yaml
```

The `values.yaml` in this repo:

- pins the operator image and resources
- enables the admission webhook
- widens `watchNamespaces` to include both `default` and `flink-jobs` so the
  operator reconciles `FlinkDeployment` resources outside its own namespace
- tunes reconcile/progress intervals and enables the SLF4J metrics reporter

Verify the operator and the CRDs installed by the chart:

```bash
kubectl -n flink-operator get pods
kubectl get crd | grep flink.apache.org
```

Expected CRDs:

```
flinkbluegreendeployments.flink.apache.org
flinkdeployments.flink.apache.org
flinksessionjobs.flink.apache.org
flinkstatesnapshots.flink.apache.org
```

### 4. Submit a sample Flink job

`flink-example-job.yaml` creates a `FlinkDeployment` that runs the built-in
`StateMachineExample.jar` shipped inside the `flink:1.20` image. It uses an
ephemeral `emptyDir` for checkpoints/savepoints (enough for a local demo).

```bash
kubectl apply -f flink-example-job.yaml
kubectl -n flink-jobs get flinkdeployment,pods -w
```

Expected final state:

```
NAME                                                     JOB STATUS   LIFECYCLE STATE
flinkdeployment.flink.apache.org/state-machine-example   RUNNING      STABLE

NAME                                         READY   STATUS    RESTARTS
pod/state-machine-example-57b596f7b9-xxxxx   1/1     Running   0          # JobManager
pod/state-machine-example-taskmanager-1-1    1/1     Running   0          # TaskManager
```

### 5. Follow the logs

**JobManager logs** (deployment) — shows scheduling, checkpoints, and the
execution graph transitioning to `RUNNING`:

```bash
kubectl -n flink-jobs logs deploy/state-machine-example --tail=50 -f
```

Observed in this run:

```
Source: Events Generator Source (1/2) switched from INITIALIZING to RUNNING.
Flat Map -> Sink: Print to Std. Out (1/2) switched from INITIALIZING to RUNNING.
Triggering checkpoint 1 ... for job 71fcf6e02fee8417e211b6411bd8384e.
Completed checkpoint 1 (2802 bytes, checkpointDuration=944 ms).
Completed checkpoint 2 (16617 bytes, checkpointDuration=22 ms).
Completed checkpoint 3 (15618 bytes, checkpointDuration=10 ms).
Completed checkpoint 4 (15888 bytes, checkpointDuration=11 ms).
```

**TaskManager logs** — shows the actual data processing tasks going live:

```bash
kubectl -n flink-jobs logs state-machine-example-taskmanager-1-1 --tail=50 -f
```

**Operator logs** — shows reconciliation converging and the resource reaching
`STABLE`:

```bash
kubectl -n flink-operator logs deploy/flink-kubernetes-operator -c flink-kubernetes-operator -f
```

Observed audit events:

```
Event[Job]  | Info | JOBSTATUSCHANGED | Job status changed from CREATED to RUNNING
Status[Job] | Info | STABLE           | The resource deployment is considered to be stable ...
Resource fully reconciled, nothing to do...
```

### 6. (Optional) Access the Flink UI

```bash
kubectl -n flink-jobs port-forward svc/state-machine-example-rest 8081:8081
# Then open http://localhost:8081
```

### 7. Cleanup

```bash
kubectl delete -f flink-example-job.yaml
helm uninstall flink-kubernetes-operator -n flink-operator
kubectl delete -f https://github.com/cert-manager/cert-manager/releases/download/v1.20.2/cert-manager.yaml
kind delete cluster --name flink-lab
```

---

## Scaling & security on Kubernetes — notes from heavy workloads

The toy `StateMachineExample` in this PoC barely scratches the surface of
what Flink is usually asked to do in production (clickstreams, fraud
detection, CDC pipelines, feature stores, IoT telemetry — often in the
hundreds of thousands to millions of events per second). The two topics
that dominate every production conversation about Flink-on-Kubernetes are
**scaling** and **security**. Here is a brief based on the upstream
documentation and write-ups from teams running Flink at scale.

### Scaling

**Vertex-level autoscaling is the killer feature.** The Flink Kubernetes
Operator ships a built-in
[Job Autoscaler](https://nightlies.apache.org/flink/flink-kubernetes-operator-docs-main/docs/custom-resource/autoscaler/)
that collects per-operator metrics (true processing rate, busy time,
backpressure, backlog growth, lag) and scales **individual job vertices**
independently — not just global job parallelism. This matters enormously
for real pipelines because large streaming jobs are almost always
*heterogeneous*: a cheap source, one very hot keyed aggregation, a network
shuffle, and an expensive sink. Scaling the whole job to the worst vertex
wastes resources everywhere else. FLIP-271 codified this model and it is
now the standard approach on Kubernetes.

**In-place rescaling (Flink 1.18+) drastically reduces the cost of a scale
event.** Historically every rescale meant: take a savepoint, stop the
job, relaunch the JobGraph with new parallelism, restore. The new
[Resource Requirements REST endpoint](https://nightlies.apache.org/flink/flink-docs-master/docs/deployment/elastic_scaling/)
lets the autoscaler change vertex parallelism without a full upgrade
cycle, so scaling goes from "minute-scale downtime" to "seconds" for
jobs in Reactive / Adaptive Scheduler mode. Teams running heavy,
always-on jobs cite this as the reason they finally trusted the
autoscaler in production.

**The autoscaler is explicitly designed for heavy workloads, not toy
jobs.** The upstream docs state the autoscaler is "geared towards
operations processing substantial data volumes" — if you only need a
couple of TaskManagers, you're better off pinning parallelism. The
metrics the algorithm cares about are throughput, **backlog growth**,
**backlog time**, and **CPU utilization**; these are the signals that
tell you whether a streaming job is actually keeping up with the
firehose, not just whether a pod is busy.

**Every scale event has a cost — the algorithm must know it.** Unlike a
stateless HTTP service, Flink carries state. Even with in-place
rescaling, a scale-up/down triggers state redistribution across key
groups, re-hashing, and checkpoint churn. The autoscaler explicitly
models a "scaling cost" so it doesn't thrash. In practice this means
teams tune `stabilization-interval`, `scale-down-factor`, and
`target-utilization` conservatively for stateful, high-throughput jobs.

**Spot/preemptible nodes are table stakes for cost.** AWS's own write-up
on running Flink on EKS with EC2 Spot shows that Flink's checkpointing
model maps naturally to preemptible capacity: TaskManagers on Spot,
JobManager on on-demand, Pod Disruption Budgets to bound concurrent
evictions, and the operator's HA mode to recover from JobManager
rescheduling. Combined with the autoscaler, this is how teams get Flink
bills down by 50–70% versus a statically-sized cluster.

**Plain HPA is not enough on its own.** Horizontal Pod Autoscaler scales
pods on CPU/memory, but it has no idea what Flink's *logical*
parallelism is, and it cannot trigger savepoint-aware restarts. Teams
that tried HPA-only setups (Medium write-ups from practitioners are full
of these stories) eventually moved to the operator's autoscaler or
Reactive Mode because HPA alone caused state loss or endless backpressure
storms. HPA still has a role — but it scales the *node pool*, not the
Flink job.

### Security

**Two RBAC identities, never one.** The operator's
[RBAC model](https://nightlies.apache.org/flink/flink-kubernetes-operator-docs-main/docs/operations/rbac/)
deliberately separates the `flink-operator` ServiceAccount (cluster-wide,
reconciles `FlinkDeployment` CRs, creates JobManager deployments) from
the per-job `flink` ServiceAccount (namespace-scoped, used by the
JobManager to create TaskManagers and ConfigMaps). Collapsing these into
one SA is a common mistake and it gives any compromised JobManager the
operator's privileges over the whole cluster.

**Do not hand users raw access to the underlying Kubernetes resources.**
The upstream guidance is explicit: users should interact with
`FlinkDeployment` / `FlinkSessionJob` CRs, not with the JobManager
Deployments, Services or ConfigMaps the operator generates. Granting
`get pods` + `exec` on a JobManager pod is effectively a shell into the
JVM running user-provided JAR code — which typically has the job's cloud
credentials mounted.

**Per-job ServiceAccount enforcement (least privilege).** Vendors like
Ververica explicitly recommend that *each Flink deployment runs under its
own ServiceAccount*, so a compromised or buggy job cannot assume the
privileges of another. On AWS this is IRSA per-job; on GCP it is
Workload Identity per-job; on-prem it is one SA + one Role per
`FlinkDeployment`. This is the single highest-leverage hardening step
for multi-tenant Flink clusters.

**Encrypt everything on the wire.** Flink supports TLS/SSL for both
*internal* traffic (JobManager ↔ TaskManager, blob server, RPC) and
*external* traffic (REST API, web UI), with mTLS when you need mutual
authentication. On Kubernetes this pairs naturally with cert-manager
(which we already installed in this PoC for the operator webhook) —
issue a `Certificate` per FlinkDeployment, mount it as a Secret, and
point `security.ssl.*` at it.

**Lock down the REST/UI endpoint.** The Flink web UI exposes a
`/jars/upload` endpoint by default — in session mode this is a remote
code execution primitive. Production deployments either disable JAR
upload (`web.submit.enable: false`, `web.cancel.enable: false`), run
Application mode (where there *is* no shared session cluster to upload
into), or front the UI with an authenticating proxy + NetworkPolicy so
only platform engineers can reach it.

**Standard Kubernetes hardening still applies.** NSA/CISA's
[Kubernetes Hardening Guide](https://media.defense.gov/2022/Aug/29/2003066362/-1/-1/0/CTR_KUBERNETES_HARDENING_GUIDANCE_1.2_20220829.PDF)
and Kubernetes' own
[RBAC Good Practices](https://kubernetes.io/docs/concepts/security/rbac-good-practices/)
— non-root containers, read-only root FS, seccomp `RuntimeDefault`,
PodSecurity `restricted` where possible, NetworkPolicies between the
`flink-operator` and `flink-jobs` namespaces, ResourceQuotas to cap
runaway autoscaling, image provenance via signed images — all of these
apply to Flink exactly like any other workload. Flink does not get a
pass just because it's stateful.

**Secrets, not env vars.** Kafka SASL passwords, S3/IAM tokens,
Schema Registry credentials — all of these should ride in as mounted
Secrets (or better, CSI-mounted short-lived credentials from a
Vault/Cloud KMS), never as plain env vars in the `FlinkDeployment` YAML.
Env vars show up in `kubectl describe`, in audit logs, and in
`/proc/1/environ` inside the container.

### Sources

- [Apache — Flink Kubernetes Operator: Autoscaler](https://nightlies.apache.org/flink/flink-kubernetes-operator-docs-main/docs/custom-resource/autoscaler/)
- [Apache — Elastic Scaling (Adaptive / Reactive)](https://nightlies.apache.org/flink/flink-docs-master/docs/deployment/elastic_scaling/)
- [FLIP-271: Autoscaling](https://cwiki.apache.org/confluence/display/FLINK/FLIP-271:+Autoscaling)
- [Apache — Scaling Flink automatically with Reactive Mode](https://flink.apache.org/2021/05/06/scaling-flink-automatically-with-reactive-mode/)
- [AWS — Optimizing Apache Flink on Amazon EKS using EC2 Spot Instances](https://aws.amazon.com/blogs/compute/optimizing-apache-flink-on-amazon-eks-using-amazon-ec2-spot-instances/)
- [Apache Beam blog — Crafting an Autoscaler for Apache Beam on Flink](https://beam.apache.org/blog/apache-beam-flink-and-kubernetes-part3/)
- [Medium — Auto-scaling Flink Pipelines with the Operator and HPA](https://medium.com/@krish.arava/auto-scaling-apache-flink-pipelines-using-kubernetes-flinkoperator-and-hpa-40b18ecaab8a)
- [Apache — Flink Kubernetes Operator: RBAC model](https://nightlies.apache.org/flink/flink-kubernetes-operator-docs-main/docs/operations/rbac/)
- [Ververica — Fine-Grained Access Control via Service Account Enforcement](https://www.ververica.com/knowledge-base/how-to-enforce-fine-grained-access-control-for-apache-flink-deployments-service-account-enforcement)
- [Confluent — Securing a Flink Job](https://docs.confluent.io/platform/current/flink/flink-jobs/security.html)
- [XenonStack — The Ultimate Guide to Apache Flink Security and Deployment](https://www.xenonstack.com/blog/apache-flink-security)
- [Kubernetes — RBAC Good Practices](https://kubernetes.io/docs/concepts/security/rbac-good-practices/)
- [NSA/CISA — Kubernetes Hardening Guide (PDF)](https://media.defense.gov/2022/Aug/29/2003066362/-1/-1/0/CTR_KUBERNETES_HARDENING_GUIDANCE_1.2_20220829.PDF)

---

## Why run Flink on Kubernetes?

Apache Flink is a distributed stream-processing engine. Running it on
Kubernetes — and specifically through the official operator — brings a
number of practical advantages over a traditional standalone or YARN
deployment:

- **Declarative, GitOps-friendly jobs.** A Flink job is just a
  `FlinkDeployment` YAML you can version, review, and `kubectl apply`. No
  more bespoke submission scripts or imperative CLI commands baked into CI.
- **Native lifecycle management.** The operator handles submission, upgrades,
  stateful savepoint/resume flows, rollbacks, suspend/resume, and HA
  JobManager failover. You describe the *desired* state; the operator
  drives the cluster toward it.
- **Elasticity and bin-packing.** TaskManagers are pods. They get scheduled,
  rescheduled, autoscaled (with the Flink autoscaler), and share node
  capacity with the rest of your workloads — no dedicated Flink cluster
  sitting idle.
- **Unified observability.** Logs go through the standard Kubernetes logging
  pipeline; metrics flow via the operator's reporters (Prometheus / SLF4J /
  etc.); events land in the Kubernetes event stream. One pane of glass for
  the whole platform.
- **Isolation and multi-tenancy.** Namespaces, RBAC, NetworkPolicies,
  ResourceQuotas and PodSecurity apply to Flink jobs the same way they
  apply to any workload. Teams can ship independent jobs without fighting
  over a shared session cluster.
- **Application mode by default.** Each job gets its own JobManager +
  TaskManagers, so noisy-neighbor issues and classloader leaks between
  unrelated jobs disappear.
- **Portability.** The same manifests work on kind, EKS, GKE, AKS,
  OpenShift, or on-prem — which is exactly why this PoC runs on a local
  kind cluster as a faithful preview of the production setup.

In short: Kubernetes gives Flink a modern, declarative control plane, and
the operator turns Flink into a first-class Kubernetes citizen instead of a
special-case cluster you have to babysit.
