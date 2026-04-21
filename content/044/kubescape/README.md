---
title: Kubescape — security scanner (manifests + cluster vivo) com frameworks
tags: [cncf, incubating, kubescape, security, scanner, compliance]
status: stable
---

## Kubescape (CNCF Incubating)

**O que é:** scanner de segurança para Kubernetes. Avalia cluster vivo **e** YAML em disco contra frameworks: **NSA-CISA**, **MITRE ATT&CK**, **ArmoBest**, **AllControls**, **cis-v1.23**, **SOC2**, **PCI**. Output texto/JSON/JUnit/HTML — plugável em CI.

**Quando usar (SRE day-to-day):**

- CI: `kubescape scan` bloqueia PR que introduz workload inseguro.
- Auditoria de cluster existente: "quantos workloads estão violando NSA framework?"
- Imagem scanning (Trivy como backend) — vulnerabilidades conhecidas.
- Relatório mensal de compliance SOC2/PCI.

**Kubescape vs Kyverno vs Gatekeeper:**
- **Kubescape**: scanner/auditoria. Não bloqueia no admission.
- **Kyverno/Gatekeeper**: admission controller. Bloqueiam antes de entrar.

Ideal: os dois. Kyverno bloqueia novo; Kubescape audita o passivo.

### Cenário real

*"Quero um comando que olhe TODOS os manifests YAML deste repo e reporte violações do NSA framework antes do merge."*

### Reproducing

```bash
cd content/044/kubescape
docker compose up
```

Deve exibir um relatório NSA com violations no `bad-deploy.yaml`:
- CVE em `nginx:latest`
- `hostNetwork: true`
- Container privileged
- `runAsUser: 0`
- Sem `resources.limits`

### Scan direto via CLI (sem Docker)

```bash
# instalar
curl -s https://raw.githubusercontent.com/kubescape/kubescape/master/install.sh | /bin/bash

# scan do cluster atual (requer kubeconfig válido)
kubescape scan --enable-host-scan

# scan de manifests locais
kubescape scan framework nsa ./manifests/ --format json > report.json
```

### CI (exemplo)

```yaml
# GitHub Actions
- uses: kubescape/github-action@v1
  with:
    frameworks: "nsa,mitre"
    failure-threshold: medium
```

### Cleanup

```bash
docker compose down -v
```

### Tips de SRE

- **Kubescape Operator** (in-cluster) faz scan contínuo + expor métricas Prometheus + integra com ArmoSecurity para dashboards.
- **SBOM** e vulnerability scanning: usa Trivy como backend. `kubescape scan image <img>`.
- **Baseline**: rode na primeira vez, anote violações inevitáveis (legacy), rode depois só sobre o delta.
- **Exceptions**: `kubescape scan --exceptions` com arquivo — formalize o que é aceito.
- **Priorização**: comece pelos Critical. Low pode virar backlog.

### References

- https://github.com/kubescape/kubescape
- https://hub.armosec.io/docs
