---
title: Backstage — developer portal (catálogo + TechDocs + templates)
tags: [cncf, incubating, backstage, developer-portal, internal-tools]
status: stable
---

## Backstage (CNCF Incubating)

**O que é:** plataforma de developer portal criada pelo Spotify. Integra em um só lugar:

- **Software Catalog** — mapa de todos os services, owners, deps, SLOs.
- **TechDocs** — docs-as-code (Markdown no repo → publicado no portal).
- **Scaffolder** — templates de "bootstrap" (dev clica, ganha repo + CI + deploy pipeline prontos).
- **Plugins** — 150+ (Jenkins, Jira, PagerDuty, Grafana, ArgoCD, Kubernetes, GitHub, etc) integram no mesmo pane.

**Quando usar (SRE day-to-day):**

- Onboard de novo dev ("onde fica o service de checkout? quem é owner? como deployo?") vira 1 link.
- Fim do "tribal knowledge" — ownership, deps, maturity lá, todos veem.
- Self-service — dev cria novo microserviço do template, não precisa chamar SRE.
- Single pane of glass para produção (links para Grafana/ArgoCD/Jira do service).

**Quando NÃO usar:**

- Time <15 pessoas — overhead operacional grande.
- Sem commit cultural — portal virgem vira tumba.

### Cenário real

*"Temos 200 microsserviços. Ninguém sabe quem é dono de cada. Deploy de onboarding demora 2 semanas. Quero 1 UI central."*

### Reproducing

```bash
cd content/044/backstage

# Opção rápida: imagem oficial com catálogo demo
docker compose up -d
sleep 60  # backend bun leva ~30-60s p/ migrate+start

# UI
open http://localhost:7007
```

### Opção completa (scaffolding local)

A instalação real é via `npx @backstage/create-app`, que scaffolda um monorepo customizado, com seu PostgreSQL, autenticação (GitHub/Google/OIDC), e depois deploya via Helm chart oficial no Kubernetes.

```bash
npx @backstage/create-app@latest
cd my-backstage
yarn dev  # http://localhost:3000
```

### Catálogo (YAML-first)

```yaml
# exemplo de catalog-info.yaml em um service repo
apiVersion: backstage.io/v1alpha1
kind: Component
metadata:
  name: checkout
  annotations:
    github.com/project-slug: company/checkout
    pagerduty.com/service-id: PXXXXX
    grafana/dashboard-selector: "tags.folder == 'checkout'"
spec:
  type: service
  lifecycle: production
  owner: team-payments
  system: commerce
  providesApis: [checkout-api]
  dependsOn: [resource:postgres-checkout]
```

Backstage descobre esse YAML via GitHub Discovery, popula o catálogo, linka Grafana dashboards, PagerDuty, etc.

### Cleanup

```bash
docker compose down -v
```

### Tips de SRE

- **TechDocs** renderiza MkDocs + tema Backstage. Doc do service fica no repo; sempre atualizada.
- **Software Templates**: `template.yaml` gera repo + README + CI pipeline + ArgoCD Application. Service novo em 5min.
- **Kubernetes plugin**: mostra pods/deployments do service direto na página dele — debug sem kubeconfig.
- **Ownership obrigatório** (`spec.owner`): força cultura de "esse service é do time X".
- **Dashboard health**: adicione Grafana dashboard + Lighthouse score como signal no catálogo.

### References

- https://backstage.io/docs/
- https://github.com/backstage/backstage
- https://backstage.io/plugins
