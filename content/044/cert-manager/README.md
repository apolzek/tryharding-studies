---
title: cert-manager — ciclo de vida de TLS em Kubernetes
tags: [cncf, graduated, tls, cert-manager, letsencrypt, pki]
status: stable
---

## cert-manager (CNCF Graduated)

**O que é:** controller que automatiza emissão e renovação de certificados X.509 no Kubernetes. Suporta Let's Encrypt (ACME), Vault, Venafi, self-signed, CA privada.

**Quando usar (SRE day-to-day):**

- Let's Encrypt automático em ingress (`cert-manager.io/cluster-issuer: letsencrypt-prod` como annotation no Ingress).
- PKI interna — todo pod com mTLS sem ter que rodar OpenSSL em shell script.
- Rotação automática — `renewBefore: 720h` renova 30 dias antes do expiry. Zero paging às 3 da manhã por cert vencido.
- mTLS entre serviços (emite client + server certs via mesma CA).

**Quando NÃO usar:**

- Se já tem service mesh fazendo mTLS automático (Istio/Linkerd injetam seus próprios certs) — aí cert-manager é só para o ingress edge.
- Para secrets genéricos (não-TLS) use sealed-secrets / external-secrets.

### Cenário real

*"Cada deploy de microsserviço precisa de um cert para mTLS assinado pela CA interna do cluster, com rotação diária. Não quero humano envolvido."*

Este POC cria um `ClusterIssuer` selfSigned e emite um `Certificate` que é materializado em um Secret `demo-tls-secret` — pronto para montar em pods ou Ingresses.

### Reproducing

```bash
cd content/044/cert-manager

# 1. Cluster
kind create cluster --config kind.yaml

# 2. Instala cert-manager
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.16.1/cert-manager.yaml
kubectl -n cert-manager wait --for=condition=available --timeout=5m deploy --all

# 3. Cria Issuer + Certificate
kubectl apply -f issuer.yaml

# 4. Observa
kubectl get clusterissuer
kubectl -n default get certificate,certificaterequest,order,challenge 2>/dev/null
kubectl -n default get secret demo-tls-secret
```

### Inspecionando o cert emitido

```bash
kubectl -n default get secret demo-tls-secret -o jsonpath='{.data.tls\.crt}' \
  | base64 -d | openssl x509 -text -noout | head -25
```

Deve mostrar: `Subject: O=demo, CN=demo.local`, `X509v3 Subject Alternative Name: DNS:demo.local, DNS:api.demo.local`.

### Cleanup

```bash
kind delete cluster --name certmgr-poc
```

### Tips de SRE

- **Em produção**: use `ClusterIssuer` tipo ACME (Let's Encrypt) com `solver: http01` (ingress) ou `dns01` (wildcard).
- **Rate limits do LE**: 50 certs/domain/semana. Se estiver testando, use `letsencrypt-staging` primeiro.
- **Monitoring**: métricas `certmanager_certificate_expiration_timestamp_seconds` no Prometheus — alerte com `< 14d`.
- **Propagation DNS01**: se usa DNS provider (Cloudflare/Route53), entregue RBAC mínimo (só `_acme-challenge.*`).
- **Ingress annotation** (nginx-ingress): `cert-manager.io/cluster-issuer: letsencrypt-prod` + `tls:` no spec → cert é criado sozinho.

### References

- https://cert-manager.io/docs/
- https://cert-manager.io/docs/configuration/acme/ (Let's Encrypt setup)
