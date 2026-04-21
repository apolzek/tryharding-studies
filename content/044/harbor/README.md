---
title: Harbor — registry corporativo com scan + RBAC + replication
tags: [cncf, graduated, registry, harbor, supply-chain]
status: stable
---

## Harbor (CNCF Graduated)

**O que é:** registry OCI self-hosted enterprise. Wrap sobre `distribution/distribution` (ex-docker-registry) com: UI, RBAC (projetos/users/robots), image scanning (Trivy), signing (Cosign/Notary v2), replication entre registries, garbage collection, webhooks, quotas.

**Quando usar (SRE day-to-day):**

- **Air-gap / on-prem** — cluster não pode puxar do Docker Hub/ECR direto (compliance, network).
- **Mirror de upstream** — Harbor faz replication de `docker.io/*` → `harbor.internal/dockerhub-proxy/*`. Você controla cache e evita rate-limit do Hub.
- **Scan obrigatório** — política "imagem com CVE Critical não pode ser puxada em prod" via Harbor + Trivy + webhook.
- **Signing** — todo push vira Cosign signature; admission controller no cluster rejeita imagens não-assinadas.

**Quando NÃO usar:**

- Managed registry resolve (ECR, Artifact Registry, ACR) sem mantê-lo — use se cloud-native e sem compliance peculiar.
- Se só precisa de um registry local de dev, `registry:2` é 40x mais leve.

### Cenário real

*"Quero um registry interno onde CI dá push, imagens são escaneadas por Trivy, e deploy só acontece se o scan passar. Dev vê tudo via UI."*

Este POC instala Harbor no kind via Helm com expose por NodePort (para não precisar de TLS no POC). Em produção: TLS obrigatório + storage persistente + Trivy ligado.

### Reproducing

```bash
cd content/044/harbor

# 1. Cluster (com port forward 30080 p/ NodePort)
kind create cluster --config kind.yaml

# 2. Install via Helm
helm repo add harbor https://helm.goharbor.io
helm repo update
helm install harbor harbor/harbor --version 1.15.1 -f values.yaml \
  --create-namespace -n harbor --wait --timeout 10m

# 3. Abra http://localhost:30080
# user: admin
# pass: Harbor12345
```

### Push de imagem (teste real)

```bash
# em outro terminal, fazendo login como admin
docker login http://localhost:30080 -u admin -p Harbor12345

# tag + push
docker pull alpine:3.20
docker tag alpine:3.20 localhost:30080/library/alpine:3.20
docker push localhost:30080/library/alpine:3.20

# veja na UI em Projects → library → alpine
```

### Cleanup

```bash
kind delete cluster --name harbor-poc
```

### Tips de SRE

- **Storage** em prod: S3/GCS/Azure Blob via `persistence.imageChartStorage`. Evite PVCs locais.
- **Replication** policy: `docker.io/library/nginx:*` → `harbor.internal/dockerhub-proxy/library/nginx:*`. Dev sempre puxa via Harbor.
- **Robot accounts** para CI (não use admin). Escopo read/write por projeto.
- **Garbage collection**: habilite schedule semanal; storage vaza rápido com deleções lógicas.
- **Trivy** (desligado neste POC para velocidade): em prod, ligue e configure `severity: CRITICAL,HIGH` como bloqueante em policies.
- **Webhook** p/ Slack quando scan falha ou replication errou.

### References

- https://goharbor.io/docs/
- https://artifacthub.io/packages/helm/harbor/harbor
