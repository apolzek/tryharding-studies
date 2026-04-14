## Blackbox exporter on Kind monitored by kube-prometheus-stack

### Objectives

Stand up a local Kind cluster on the newest stable Kubernetes (v1.33.1) and
wire a full blackbox probing pipeline on top of it. The Prometheus Operator
shipped by `kube-prometheus-stack` is the operator that manages the blackbox
integration: it watches `Probe` CRDs and generates the scrape config that
makes Prometheus hit the `prometheus-blackbox-exporter` `/probe` endpoint for
each target. Register a handful of HTTP, TCP and DNS targets, confirm the
metrics flow end-to-end and visualise them in the Grafana instance that ships
inside the stack.

### Prerequisites

- `kind` >= 0.24
- `kubectl` >= 1.33
- `helm` >= 3.15
- Docker running

### Reproducing

```bash
chmod +x install.sh test.sh
./install.sh        # create cluster + install everything
./test.sh           # sanity-check probes end-to-end
```

What `install.sh` does:

1. Creates a 3-node Kind cluster pinned to `kindest/node:v1.33.1` with
   NodePorts mapped to the host (`30090` Prometheus, `30300` Grafana).
2. Installs `kube-prometheus-stack` with `values-kube-prometheus-stack.yaml`.
   Selectors are relaxed (`*SelectorNilUsesHelmValues: false`) so Prometheus
   discovers any `Probe` / `ServiceMonitor` in any namespace.
3. Installs `prometheus-blackbox-exporter` with `values-blackbox.yaml` which
   declares the `http_2xx`, `http_post_2xx`, `tcp_connect`, `icmp` and
   `dns_google` modules.
4. Applies `probes.yaml` — three `Probe` CRDs registering HTTP, TCP and DNS
   targets.
5. Applies `blackbox-dashboard-cm.yaml` — a `ConfigMap` holding a custom
   *Blackbox Exporter* dashboard (`blackbox-dashboard.json`, 7 panels) that
   targets the Prometheus datasource by **UID** (`{type: prometheus, uid:
   prometheus}`) — the format Grafana 12 actually understands. The CM
   carries the `grafana_dashboard=1` label so the Grafana sidecar auto-loads
   it at startup with no manual import step.

   **Why a custom dashboard?** The popular grafana.com dashboard 7587 dates
   from 2018 and references its datasource as the legacy string
   `"Prometheus"`. Grafana 12 resolves datasources by UID and silently
   shows every panel as `No data` / `PanelQueryRunner Error` when it can't
   resolve the legacy string. The custom JSON is minimal, schema-version 39,
   and every `datasource` block (dashboard-level, panel-level, target-level
   and variable-level) points at the UID directly — verified rendering on a
   fresh Grafana 12.4.3.

### Accessing Grafana

Two options — the stack exposes both simultaneously:

```bash
# Option A — NodePort exposed by Kind's extraPortMappings
open http://localhost:30300

# Option B — kubectl port-forward (matches the session this POC was built in)
kubectl -n monitoring port-forward svc/kps-grafana 3000:80
open http://127.0.0.1:3000
```

Login: `admin` / `admin`. The blackbox dashboard is available at
`/d/xtkCtBkiz/blackbox-exporter` on whichever host/port you chose.

If the initial login rejects `admin/admin` (the Grafana secret is
re-generated on every reinstall), reset it in-cluster:

```bash
kubectl -n monitoring exec deploy/kps-grafana -c grafana -- \
  grafana-cli --homepath /usr/share/grafana admin reset-admin-password admin
```

### Targets registered

| Kind | Module | Targets |
|---|---|---|
| HTTP | `http_2xx` | google.com, github.com, prometheus.io, kubernetes.io, grafana.com |
| TCP  | `tcp_connect` | github.com:22, github.com:443, 1.1.1.1:53 |
| DNS  | `dns_google` | 8.8.8.8, 1.1.1.1 |

### Results

Verified end-to-end on a real run of `install.sh` (Kind v1.33.1 on Ubuntu
24.04 / Docker 29.4.0, kube-prometheus-stack latest, prometheus-blackbox-exporter
latest).

Direct probe against the exporter service:

```
$ curl 'http://blackbox…:9115/probe?target=https://www.google.com&module=http_2xx'
probe_duration_seconds 0.299789052
probe_http_status_code 200
probe_ssl_earliest_cert_expiry 1.781512792e+09
probe_success 1
```

`probe_success` series in Prometheus after reconcile (all UP):

```
job=blackbox-http-public  target=https://www.google.com        UP
job=blackbox-http-public  target=https://www.github.com        UP
job=blackbox-http-public  target=https://prometheus.io         UP
job=blackbox-http-public  target=https://kubernetes.io         UP
job=blackbox-http-public  target=https://grafana.com           UP
job=blackbox-tcp          target=github.com:22                 UP
job=blackbox-tcp          target=github.com:443                UP
job=blackbox-tcp          target=1.1.1.1:53                    UP
job=blackbox-dns          target=8.8.8.8                       UP
job=blackbox-dns          target=1.1.1.1                       UP
```

`probe_http_status_code` returns `200` for all five HTTP targets.

UIs:

- Prometheus UI: <http://localhost:30090> → *Status → Targets* — the three
  `probe/monitoring/*` scrape pools appear and are green.
- Grafana: NodePort <http://localhost:30300> or port-forward
  <http://127.0.0.1:3000> (`admin` / `admin`). The custom *Blackbox
  Exporter* dashboard is pre-loaded at
  `/d/xtkCtBkiz/blackbox-exporter` with 7 panels — Targets UP, Targets
  DOWN, Avg probe duration, Min SSL cert days remaining, a per-target
  status table and two timeseries (probe duration, probe success). A
  live query through the Grafana datasource proxy
  (`/api/datasources/proxy/uid/prometheus/api/v1/query?query=probe_success`)
  returned 10 series, all value `1`, confirming the panels render with data
  end-to-end. Sidecar log on load:
  `Writing /tmp/dashboards/blackbox.json` →
  `POST /api/admin/provisioning/dashboards/reload → 200 OK`.

Useful PromQL:

- `probe_success` — 1 if the probe passed, 0 otherwise.
- `probe_duration_seconds` — end-to-end probe latency.
- `probe_http_status_code` — HTTP response code for HTTP probes.
- `probe_ssl_earliest_cert_expiry - time()` — seconds until cert expiry.

### Gotcha

On this chart version the Prometheus Operator does **not** interpret an empty
`probeSelector: {}` as "match everything". Leaving it empty produces zero
`probe/*` scrape jobs even though the `Probe` CRDs exist. The values file
therefore pins `probeSelector.matchLabels.release: kps` (and the same for
`serviceMonitorSelector`, `podMonitorSelector`, `ruleSelector`), and every
`Probe` in `probes.yaml` carries the `release: kps` label. If you drop that
label or change the selector back to `{}`, targets will silently disappear.

### Prerequisite tweak on Ubuntu 24.04

Kind multi-node clusters on Ubuntu 24.04 crash at boot with
`Failed to create control group inotify object: Too many open files`. Raise
the inotify limits once before `install.sh`:

```bash
sudo sysctl -w fs.inotify.max_user_instances=8192 \
                fs.inotify.max_user_watches=524288
```

Tear-down:

```bash
kind delete cluster --name blackbox-lab
```

### References

- <https://github.com/prometheus-operator/kube-prometheus>
- <https://github.com/prometheus-community/helm-charts/tree/main/charts/kube-prometheus-stack>
- <https://github.com/prometheus-community/helm-charts/tree/main/charts/prometheus-blackbox-exporter>
- <https://prometheus-operator.dev/docs/api-reference/api/#monitoring.coreos.com/v1.Probe>
- <https://grafana.com/grafana/dashboards/7587-prometheus-blackbox-exporter/>
- <https://kind.sigs.k8s.io/>
