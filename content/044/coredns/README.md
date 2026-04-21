---
title: CoreDNS — DNS plugável para resolução interna/split-horizon
tags: [cncf, graduated, dns, coredns, networking]
status: stable
---

## CoreDNS (CNCF Graduated)

**O que é:** servidor DNS em Go, arquitetura plugin-based. É o DNS default do Kubernetes (`kube-dns` de fato é CoreDNS desde v1.13). Pode rodar standalone também.

**Quando usar (SRE day-to-day):**

- DNS do cluster Kubernetes (já vem, mas você customiza com ConfigMap).
- **Split-horizon** DNS — `prod.internal` resolve diferente em dev vs prod.
- Override de domínio interno — `artifact.company.internal` aponta para o Harbor privado.
- Forwarding seletivo — `*.aws.internal` vai para Route53 resolver, resto para 1.1.1.1.
- Métricas Prometheus DNS nativas.

### Cenário real

*"Minha rede interna usa `.internal.lab` e `.internal` como zonas privadas. Quero resolver essas localmente e mandar o resto para DNS público, com cache de 30s e métricas Prometheus."*

### Reproducing

```bash
cd content/044/coredns
docker compose up -d
sleep 3
```

Teste resolução:

```bash
# zona por file plugin
dig @127.0.0.1 -p 1053 api.internal.lab +short
dig @127.0.0.1 -p 1053 db.internal.lab +short
dig @127.0.0.1 -p 1053 cache.internal.lab +short  # CNAME → db

# zona por hosts plugin
dig @127.0.0.1 -p 1053 payments.internal +short

# forward para upstream (qualquer domínio não-local)
dig @127.0.0.1 -p 1053 cloudflare.com +short

# métricas Prometheus
curl -s http://localhost:9153/metrics | grep -E "^coredns_dns_" | head -10
```

### Cleanup

```bash
docker compose down -v
```

### Tips de SRE

- **Plugin order matters**. `cache` antes de `forward` = cache hit não sai para upstream. Errado: cache depois = cada query vai para upstream mesmo com TTL.
- **Autopath** (k8s) reduz queries negativas de `service.namespace.svc.cluster.local` até resolver — economia enorme em clusters busy.
- **Prometheus**: scrape `:9153` — alerte em `coredns_dns_request_duration_seconds` alto (latência de DNS trava apps inteiras).
- **Forwarder redundante**: use `forward . 8.8.8.8 1.1.1.1` + `health_check 5s` — se o primário cair, cai no secundário sem timeout.
- **DNS over TLS/HTTPS**: `forward . tls://1.1.1.1` se quer privacidade upstream.

### References

- https://coredns.io/plugins/
- https://coredns.io/manual/toc/
