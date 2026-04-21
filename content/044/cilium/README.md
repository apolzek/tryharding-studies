---
title: Cilium — CNI baseado em eBPF (networking, observabilidade, policy)
tags: [cncf, graduated, cilium, ebpf, cni, networking, hubble]
status: stable
---

## Cilium (CNCF Graduated)

**O que é:** CNI para Kubernetes construído em cima de **eBPF** — substitui iptables/kube-proxy por programas eBPF no kernel. Inclui networking (pod-to-pod, service), L3/L4/L7 NetworkPolicy (CiliumNetworkPolicy), load-balancing, observabilidade (Hubble), cluster mesh, service mesh sem sidecar.

**Quando usar (SRE day-to-day):**

- Substituir `kube-proxy` + iptables em clusters grandes (performance e escalabilidade).
- **NetworkPolicy L7** — política baseada em HTTP path/method (`GET /api/*`), DNS, Kafka, gRPC. NetworkPolicy "normal" é só L3/L4.
- **Hubble** — observabilidade de rede por fluxo (ver quem fala com quem em tempo real, sem tcpdump).
- **Cluster Mesh** — conecta múltiplos clusters como se fossem um (serviços visíveis cross-cluster).
- Service Mesh sem sidecar (Ambient-like) — menos overhead que Istio tradicional.

**Quando NÃO usar:**

- Kernel antigo (<5.4) — eBPF precisa de features modernas.
- Se você só precisa de pod networking simples e já usa kindnet/flannel, Cilium é overkill.

### Cenário real

*"Quero bloquear tudo que não tenha label `role=client` de falar com meu deploy `web`, e visualizar em tempo real quem está sendo bloqueado."*

Este POC sobe um kind **sem CNI e sem kube-proxy**, instala Cilium via CLI, deploya 2 pods (web + client com label / attacker sem label) e aplica uma `NetworkPolicy` que só permite `role=client`.

### Reproducing

```bash
cd content/044/cilium

# 1. kind SEM CNI (Cilium será o CNI)
kind create cluster --config kind.yaml

# 2. Cilium CLI
CILIUM_CLI_VERSION=$(curl -s https://raw.githubusercontent.com/cilium/cilium-cli/main/stable.txt)
curl -L --fail --remote-name-all \
  https://github.com/cilium/cilium-cli/releases/download/${CILIUM_CLI_VERSION}/cilium-linux-amd64.tar.gz
tar -xzf cilium-linux-amd64.tar.gz && rm cilium-linux-amd64.tar.gz
./cilium install --version 1.16.3

# 3. Wait ready
./cilium status --wait

# 4. Workloads de teste
kubectl create deploy web --image=nginx --port=80
kubectl expose deploy web --port=80
kubectl run client --image=curlimages/curl:8.10.1 --labels=role=client \
  --command -- sleep 3600
kubectl run attacker --image=curlimages/curl:8.10.1 \
  --command -- sleep 3600
kubectl wait --for=condition=ready pod --all --timeout=2m

# 5. Antes da policy: ambos conseguem
kubectl exec client   -- curl -sI http://web | head -1   # 200 OK
kubectl exec attacker -- curl -sI http://web | head -1   # 200 OK

# 6. Aplica policy
kubectl apply -f netpol.yaml

# 7. Depois: attacker é bloqueado
kubectl exec client   -- curl -sI http://web --max-time 3 | head -1   # 200
kubectl exec attacker -- curl -sI http://web --max-time 3 | head -1   # timeout (bloqueado)
```

### Hubble (observabilidade)

```bash
./cilium hubble enable
./cilium hubble port-forward &
./cilium hubble observe --pod attacker   # mostra DROP events em tempo real
```

### Cleanup

```bash
kind delete cluster --name cilium-poc
rm -f cilium
```

### Tips de SRE

- **CiliumNetworkPolicy** (CRD) > NetworkPolicy vanilla — permite L7, DNS-based, ICMP, port ranges.
- **Hubble UI** em prod: gráfico Sankey de quem fala com quem. Troubleshoot de firewall sem shell no pod.
- **Kube-proxy replacement**: `kubeProxyMode: none` no kind (feito aqui) + `kubeProxyReplacement=true` no Cilium. Menos conntrack overhead em prod.
- **Cluster Mesh** para multi-region: os endpoints de serviço ficam acessíveis cross-cluster sem VPN.
- eBPF programs são por nó — se um nó estiver com kernel antigo, Cilium falha ali (pin node selector em pools de nós novos).

### References

- https://docs.cilium.io/
- https://docs.cilium.io/en/stable/gettingstarted/kind/
