---
title: SPIRE — identidade de workload (SPIFFE implementation)
tags: [cncf, graduated, spire, spiffe, identity, zero-trust, mtls]
status: stable
---

## SPIRE (CNCF Graduated)

**O que é:** implementação de referência do **SPIFFE** (Secure Production Identity Framework For Everyone). Entrega **identidade criptográfica** (SPIFFE ID + SVID) para cada workload — sem passwords, sem tokens compartilhados. SVIDs são X.509 certs ou JWTs com TTL curto (ex: 1h).

**SPIFFE/SPIRE em 3 frases:**
1. Cada workload tem ID único `spiffe://trust-domain/ns/foo/sa/bar`.
2. SPIRE Agent local atesta o workload (via selectors: k8s sa, unix uid, AWS IAM role...) e entrega um SVID.
3. Workload usa SVID para mTLS / JWT bearer sem nunca tocar em password.

**Quando usar (SRE day-to-day):**

- **Zero Trust** real — identidades fortes em tudo, não senha em `.env`.
- mTLS entre serviços sem cert-manager manual por app.
- Plugar em Envoy/Istio como CA (SPIRE issue os certs de mesh).
- Identity para apps heterogêneas (VM + k8s + lambda).

**Quando NÃO usar:**

- Cluster pequeno, mesh já faz mTLS (Linkerd/Istio gerenciam internamente).
- Time que ainda compartilha `.env` — SPIRE exige maturidade.

### Cenário real

*"Quero que cada pod consiga se autenticar em outro pod via mTLS sem ninguém nunca colocar `client_cert.pem` num Secret. E quero rotação automática a cada 1h."*

### Reproducing

```bash
cd content/044/spire

# 1. Cluster
kind create cluster --config kind.yaml

# 2. Install SPIRE via chart oficial
helm repo add spiffe https://spiffe.github.io/helm-charts-hardened/
helm install -n spire-server spire-crds spiffe/spire-crds \
  --create-namespace
helm install -n spire-server spire spiffe/spire \
  --version 0.24.2 \
  --set global.spire.trustDomain=poc.local \
  --wait --timeout 5m

kubectl -n spire-server get pods
kubectl -n spire-system get pods   # agents
```

### Obtendo um SVID

```bash
# Registrar um workload: pod com ServiceAccount default no ns default
kubectl apply -f - <<EOF
apiVersion: spire.spiffe.io/v1alpha1
kind: ClusterSPIFFEID
metadata:
  name: demo-app
spec:
  spiffeIDTemplate: "spiffe://poc.local/ns/{{ .PodMeta.Namespace }}/sa/{{ .PodSpec.ServiceAccountName }}"
  podSelector:
    matchLabels:
      app: demo
EOF

# sobe um pod com o label
kubectl create deploy demo --image=ghcr.io/spiffe/spiffe-helper:0.7.0
kubectl label deploy demo app=demo --overwrite

# dentro do pod o agent SPIRE monta socket em /run/spire/sockets/agent.sock
# a app chama o Workload API e pega o SVID
```

### Cleanup

```bash
kind delete cluster --name spire-poc
```

### Tips de SRE

- **Trust domain** é para a vida. Escolha algo estável: `company.com`, `prod.internal`.
- **Rotação**: SVIDs têm TTL curto (padrão 1h). Workload re-pede antes de expirar. App crasha se TTL expirou? Use biblioteca oficial (spiffe-helper / go-spiffe).
- **Federation** liga dois trust domains (ex: AWS SPIRE ↔ GCP SPIRE).
- **SPIRE Controller Manager** (CRDs `ClusterSPIFFEID`, `ClusterFederatedTrustDomain`) — declarativo. Prefira sobre `spire-server entry create` manual.
- **Plug-ins**: atestação por `k8s_psat`, `aws_iid`, `gcp_iit`, `unix:uid` — escolha por plataforma.
- Em mesh: Istio pode usar SPIRE como CA — um trust domain só, de ponta a ponta.

### References

- https://spiffe.io/
- https://spiffe.io/docs/latest/try/getting-started-k8s/
- https://artifacthub.io/packages/helm/spiffe/spire
