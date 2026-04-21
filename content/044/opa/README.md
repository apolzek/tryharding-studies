---
title: OPA — Open Policy Agent (policy-as-code em Rego)
tags: [cncf, graduated, opa, policy, rego, authorization]
status: stable
---

## OPA (CNCF Graduated)

**O que é:** engine de políticas de propósito geral. Você escreve políticas em **Rego**, envia input via HTTP/gRPC, OPA responde `allow: true/false` (+ dados). General-purpose: k8s admission (via Gatekeeper), HTTP API authz (Envoy ext_authz), Terraform, SQL, CI/CD.

**OPA vs Kyverno**: Kyverno é Kubernetes-only (YAML pattern-match). OPA funciona em **qualquer lugar** que fale HTTP + JSON.

**Quando usar (SRE day-to-day):**

- Sidecar de authz pra microsserviço (a app chama OPA antes de cada request sensível).
- Envoy/Istio `ext_authz` → OPA decide allow/deny no edge.
- Policy em Terraform (`opa eval` no plan → bloqueia infra ruim antes do apply).
- Gatekeeper (admission) no Kubernetes.

### Cenário real

*"Meu backend precisa perguntar para um serviço central: 'usuário X pode ler o pedido Y?'. Quero centralizar essa lógica em Rego, não espalhada por 20 repos."*

### Reproducing

```bash
cd content/044/opa
docker compose up -d
sleep 3

# admin → allow
curl -s -X POST http://localhost:8181/v1/data/authz/allow \
  -H 'Content-Type: application/json' \
  -d '{"input": {"subject":{"id":"u1","role":"admin"}, "action":"read", "resource":{"type":"order","id":"o1","owner":"u2"}}}'
# → {"result":true}

# user u1 tentando ler order do u2 → deny
curl -s -X POST http://localhost:8181/v1/data/authz/allow \
  -d '{"input": {"subject":{"id":"u1","role":"user"}, "action":"read", "resource":{"type":"order","id":"o1","owner":"u2"}}}'
# → {"result":false}

# user u1 lendo seu próprio order → allow
curl -s -X POST http://localhost:8181/v1/data/authz/allow \
  -d '{"input": {"subject":{"id":"u1","role":"user"}, "action":"read", "resource":{"type":"order","id":"o1","owner":"u1"}}}'
# → {"result":true}

# decision com motivo (ajuda auditoria)
curl -s -X POST http://localhost:8181/v1/data/authz \
  -d '{"input": {"subject":{"id":"u1","role":"user"}, "action":"read", "resource":{"type":"order","id":"o1","owner":"u2"}}}'
# → {"result":{"allow":false,"reason":"denied: not owner and not admin"}}
```

### Cleanup

```bash
docker compose down -v
```

### Tips de SRE

- **Decision logs**: `--set=decision_logs.console=true` emite cada decision — mande para SIEM/audit log.
- **Bundles**: em prod, OPA puxa policy de um HTTP endpoint (`--bundle`) com assinatura — nunca policy no filesystem do pod.
- **Performance**: OPA é microssegundo-scale em decisões. Se ficar lento, olhar indexação da data (`bundles.status`).
- **Testes unitários**: Rego tem `*_test.rego` — rode `opa test` no CI.
- **Sidecar pattern**: OPA junto com a app (low latency). Central OPA = risco de SPOF.

### References

- https://www.openpolicyagent.org/docs/latest/
- https://play.openpolicyagent.org/ (Rego REPL online)
