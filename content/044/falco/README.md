---
title: Falco — runtime security (syscalls suspeitas → alerta)
tags: [cncf, graduated, falco, security, runtime, ebpf]
status: stable
---

## Falco (CNCF Graduated)

**O que é:** runtime security engine. Lê eventos do kernel (syscalls, k8s audit log, container events) via eBPF ou kmod. Casa contra regras (`.yaml`) e dispara alertas: "container abriu shell", "alguém leu /etc/shadow", "cryptominer conhecido", "hostNetwork em pod novo".

**Quando usar (SRE day-to-day):**

- DaemonSet em todos os nós prod — alertas realtime no Slack/PagerDuty quando algo suspeito roda.
- Evidência forense — Falco logs num SIEM (Splunk/Elastic/Loki) para investigação de incidente.
- Detecção de cryptomining, reverse shells, privilege escalation.
- Política de conformidade (PCI-DSS, HIPAA) — 160+ regras oficiais cobrem MITRE ATT&CK.

**Quando NÃO usar:**

- Se você precisa **bloquear** (admission-level) — Falco só **detecta**. Use Kyverno/Gatekeeper para bloquear antes; Falco é a rede de pesca depois.
- Ambientes sem acesso privileged ao kernel (serverless, FaaS).

### Cenário real

*"Quero ser paged se alguém abrir shell interativo em qualquer pod de prod, ou se algo ler /etc/shadow fora de `sshd`."*

### Reproducing

```bash
cd content/044/falco
docker compose up -d
sleep 6

# logs do Falco (regras carregadas + schema OK)
docker logs cncf-falco --tail 30
```

> ⚠️ **Nota sobre driver**: o POC usa `engine.kind=modern_ebpf` (kernel ≥5.8). Se seu kernel bloquear BPF (Docker Desktop no Mac/Windows, LXC confinado), troque para:
> - `falcosecurity/falco:0.39.0` (inclui kmod) rodando com `--privileged`, OU
> - deploy em cluster real via Helm `falco-security/falco`.

### Gerando eventos para o Falco detectar

```bash
# Evento 1: leitura de /etc/shadow por processo não-trusted
sudo cat /etc/shadow > /dev/null
# → "Read sensitive file untrusted" aparece em docker logs cncf-falco

# Evento 2: abrir shell em container
docker run --rm -it busybox sh -c "echo test"
# → "Terminal Shell in Container" é detectado
```

Procure nos logs:
```bash
docker logs cncf-falco 2>&1 | grep -E "Warning|Notice|Critical"
```

### Cleanup

```bash
docker compose down -v
```

### Tips de SRE

- **Output targets**: Falco pode postar em webhook (Slack/Teams), gRPC, syslog, file. Configure em `falco.yaml`.
- **Falcosidekick**: sidecar que roteia alertas para 50+ destinos (Slack, OpsGenie, Loki, ElasticSearch, S3).
- **Falco Talon**: resposta automática (kill pod, drenar nó, aplicar NetworkPolicy em caso de IoC).
- **Custom rules**: casos de negócio (`proc.name = curl and evt.arg.url contains "pastebin"`) = detecção de exfiltração.
- **eBPF modern driver**: kernel 5.8+, sem compilar kmod. Use `--modern-bpf`.
- **Volume de alertas**: Falco é verboso. Filtre `priority < WARNING` em prod, senão o SIEM afoga.

### References

- https://falco.org/docs/
- https://github.com/falcosecurity/rules (catálogo oficial)
- https://github.com/falcosecurity/falcosidekick
