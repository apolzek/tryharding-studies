---
title: Artifact Hub — catálogo central de charts, Operators, plugins etc
tags: [cncf, incubating, artifact-hub, discovery, charts]
status: stable
---

## Artifact Hub (CNCF Incubating)

**O que é:** hub web que indexa **packages cloud-native**: Helm charts, OLM/Operator Framework operators, Falco rules, OPA policies, Tekton tasks, Krew plugins, Tinkerbell actions, KEDA scalers, KubeArmor security policies, Kyverno policies, Keptn integrations. Serve como `npm search` para o ecossistema.

**Quando usar (SRE day-to-day):**

- Descobrir chart/operator/policy já pronto ao invés de escrever do zero.
- Publicar charts internos — Artifact Hub self-hosted indexa seu Harbor/GitHub.
- Verificar signature + provenance antes de instalar.
- Ver se um chart é mantido (Artifact Hub mostra **staleness**, CVEs, security audit).

### Cenário real

*"Quero um 'marketplace' interno onde devs descobrem o chart interno de redis, kafka, etc do time de plataforma. Atualmente é `git grep` em 10 repos."*

### Reproducing

**Modo 1 (recomendado):** use o Artifact Hub **público** em https://artifacthub.io — é o default. Não precisa self-hostar para procurar packages.

**Modo 2 (self-hosted):**

```bash
# Build local via docker compose do repo oficial
git clone --depth 1 https://github.com/artifacthub/hub /tmp/artifacthub
cd /tmp/artifacthub/database/migrations
docker compose up -d    # Postgres + migrations

# Ou use o chart oficial em Kubernetes
helm repo add artifact-hub https://artifacthub.github.io/helm-charts
helm install ah artifact-hub/artifact-hub \
  --set hub.ingress.enabled=false \
  -n artifact-hub --create-namespace
```

Config de um repo para tracking: via UI `/control-panel/repositories` com URL do seu Harbor / Helm repo / ChartMuseum.

### Uso na CLI (API pública)

```bash
# Procurar chart de prometheus
curl -s 'https://artifacthub.io/api/v1/packages/search?kind=0&ts_query_web=prometheus' \
  | python3 -m json.tool | head -40

# Detalhes de um package (inclui CVEs, signature, last update)
curl -s 'https://artifacthub.io/api/v1/packages/helm/prometheus-community/prometheus' \
  | python3 -c "import json,sys; d=json.load(sys.stdin); print('latest:', d['version']); print('signed:', d.get('signed',False)); print('stars:', d.get('stars',0))"
```

### Tips de SRE

- **Verified Publishers** (badge na UI) = publisher validou ownership do domínio.
- **Security Report**: Artifact Hub rola Trivy em imagens do chart — CVEs visíveis antes de `helm install`.
- **Signed** (provenance) — commit Sigstore/Cosign. Exige no chart interno também.
- **Stale** warning em pacotes sem update há muito — sinal de abandono, evite.
- **API**: integre num bot Slack que avisa "nova versão do chart X disponível".

### References

- https://artifacthub.io/
- https://github.com/artifacthub/hub
- https://artifacthub.io/docs/api/
