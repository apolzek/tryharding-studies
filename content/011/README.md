---
title: Network Connectivity Monitoring with network_exporter on Kubernetes
tags: [observability, kubernetes, networking, prometheus, network-exporter, kind]
status: stable
---

## Network Connectivity Monitoring with network_exporter on Kubernetes

### Objectives

The goal of this PoC is to monitor network connectivity from inside a Kubernetes cluster using `network_exporter`. The exporter performs active checks — ICMP ping, MTR (traceroute with stats), TCP port, and HTTP GET — against configured targets and exposes the results as Prometheus metrics. Prometheus scrapes the exporter and Grafana visualizes latency, packet loss, and reachability over time.

### Architecture

```mermaid
graph LR
    NE[network_exporter<br/>ICMP / MTR / TCP / HTTP] --:9427/metrics--> PM[Prometheus]
    PM --> GF[Grafana]
```

All components run as Kubernetes Deployments in the `default` namespace.

### Services

| Component        | Port | Kubernetes Resource                |
| ---------------- | ---- | ---------------------------------- |
| network-exporter | 9427 | Deployment + Service + ConfigMap   |
| prometheus       | 9090 | Deployment + Service + ConfigMap   |
| grafana          | 3000 | Deployment + Service + ConfigMap   |

### Prerequisites

- kubectl
- a running Kubernetes cluster (kind, minikube, or remote)

### Reproducing

Apply all manifests
```sh
kubectl apply -f network_exporter.yaml
kubectl apply -f prometheus.yaml
kubectl apply -f grafana.yaml
```

Wait for pods to be ready
```sh
kubectl get pods -w
```

Forward ports to access the UIs
```sh
kubectl port-forward svc/prometheus-service 9090:9090
kubectl port-forward svc/grafana-service 3000:80
```

Prometheus is pre-configured as a datasource in Grafana. Query metrics such as:
```
network_icmp_duration_seconds
network_mtr_rtt_seconds
network_tcp_connection_status
network_http_get_status
```

To add or change monitored targets, edit the `network_exporter.yml` section inside the ConfigMap in `network_exporter.yaml` and re-apply.

### Results

`network_exporter` covers the main active probe types needed for network visibility inside a cluster: ICMP, MTR, TCP, and HTTP. MTR metrics provide latency and path-level insights per hop, which is useful for diagnosing intermittent connectivity issues between services. The privileged container requirement (`NET_ADMIN`, `NET_RAW`) is expected for raw socket operations and needs to be accounted for in security policies. Prometheus scrapes the exporter without extra configuration, and Grafana picks up the datasource automatically via the provisioned ConfigMap.

### References

```
https://github.com/syepes/network_exporter
```

