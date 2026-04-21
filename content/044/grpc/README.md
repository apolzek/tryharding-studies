---
title: gRPC — RPC binário sobre HTTP/2 com schema (protobuf)
tags: [cncf, incubating, grpc, rpc, protobuf, microservices]
status: stable
---

## gRPC (CNCF Incubating)

**O que é:** framework de RPC criado pelo Google. Schema-first via **protobuf**, transporte **HTTP/2** (multiplexação, header compression), 4 modos: unary, server-streaming, client-streaming, bidi-streaming. SDKs em 10+ linguagens.

**Quando usar (SRE day-to-day):**

- Comunicação interna microserviço ↔ microserviço (menos overhead que REST+JSON).
- Streams (telemetria, event bus simples) — bidi é first-class.
- Compat forte via `.proto` (campos numerados, backwards compat gratuita).
- Contratos API publicados como pacote versionado (ex: internal/api-protos).

**Quando NÃO usar:**

- API pública para browser — gRPC não roda em browser diretamente. Use **gRPC-Web** (via Envoy) ou GraphQL/REST.
- Cliente humano precisa ler — JSON é mais amigável.

### Cenário real

*"Meu serviço `orders` precisa chamar `inventory` milhares de vezes/s. REST+JSON é lento (CPU de parse). Quero contratos tipados, streams e baixa latência."*

### Reproducing

Este POC sobe **grpcbin** — servidor gRPC de teste (tipo httpbin para gRPC) com reflection ligado.

```bash
cd content/044/grpc
docker compose up -d
sleep 4
```

### Chamando com grpcurl (via reflection)

```bash
# Listar serviços expostos (via reflection)
docker run --rm --network grpc_default fullstorydev/grpcurl:v1.9.1 -plaintext grpcbin:9000 list

# Describe de um service
docker run --rm --network grpc_default fullstorydev/grpcurl:v1.9.1 -plaintext grpcbin:9000 describe grpcbin.GRPCBin

# Unary call: DummyUnary ecoa payload
docker run --rm --network grpc_default fullstorydev/grpcurl:v1.9.1 -plaintext \
  -d '{"f_string": "hello gRPC", "f_int32": 42}' \
  grpcbin:9000 grpcbin.GRPCBin/DummyUnary

# Server streaming: DummyServerStream retorna 10 mensagens
docker run --rm --network grpc_default fullstorydev/grpcurl:v1.9.1 -plaintext \
  -d '{"f_string": "stream me"}' \
  grpcbin:9000 grpcbin.GRPCBin/DummyServerStream

# Health check (grpc.health.v1)
docker run --rm --network grpc_default fullstorydev/grpcurl:v1.9.1 -plaintext grpcbin:9000 grpc.health.v1.Health/Check
```

### Cleanup

```bash
docker compose down -v
```

### Tips de SRE

- **Reflection** (`grpc.reflection`): permite descobrir API em runtime. Ligue em dev/stg; considere desligar em prod por attack surface.
- **Health checking**: `grpc.health.v1.Health` — load balancer/LB tem probe padrão.
- **Retries**: cliente gRPC nativo tem retry policy via `service_config` (backoff, max attempts). Use.
- **Deadlines** obrigatórios — sem `context.WithTimeout`, request pode travar para sempre.
- **Status codes**: `UNAVAILABLE` (retry ok), `DEADLINE_EXCEEDED`, `INVALID_ARGUMENT` — alerte em % de não-OK.
- **Observabilidade**: OpenTelemetry gRPC interceptor = traces + metrics sem código custom.
- **gRPC-Web** via Envoy: browser pode falar gRPC sem `grpc-web-proxy` separado.

### References

- https://grpc.io/docs/
- https://github.com/grpc/grpc-go/tree/master/examples/route_guide
- https://github.com/fullstorydev/grpcurl
