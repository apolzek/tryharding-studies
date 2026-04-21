---
title: NATS — messaging leve (pub/sub + JetStream persistente)
tags: [cncf, incubating, nats, messaging, jetstream, pubsub]
status: stable
---

## NATS (CNCF Incubating)

**O que é:** sistema de messaging em Go, extremamente leve (~15MB binary, microseconds-latency). Suporta:

- **Core NATS**: pub/sub fire-and-forget.
- **JetStream**: persistência + replay + at-least-once (competidor de Kafka, mas leve).
- **Subject-based routing** (hierarquia tipo `orders.>`, `events.payment.*`).
- **Clustering** automático + gateways cross-region.

**NATS vs Kafka vs RabbitMQ:**

| | NATS | Kafka | RabbitMQ |
|-|------|-------|----------|
| Modelo | Subject pub/sub + JetStream | Log particionado | Queue com exchange |
| Latência | ~100µs | ~ms | ~ms |
| Footprint | 15MB | 500MB+ | 100MB+ |
| Ordenação | Por stream | Por partition | Queue FIFO |
| Uso | Edge, microsserviços, IoT | Event stream enterprise | Workflows complexos |

**Quando usar (SRE day-to-day):**

- RPC async entre microsserviços — NATS Core como bus leve.
- Event sourcing simples com JetStream — não precisa de Kafka se stream é pequeno.
- IoT / edge — NATS roda em Raspberry Pi de boa.
- **Leaf nodes**: extende cluster central com leaves em redes isoladas (NAT, vans móveis).

### Cenário real

*"Tenho 5 microsserviços que precisam trocar eventos `order.created` e `payment.processed`. Kafka é overkill, RabbitMQ exige `amqp` setup. Quero algo simples."*

### Reproducing

```bash
cd content/044/nats
docker compose up -d
sleep 4

# Status do cluster
curl -s http://localhost:8222/varz | python3 -m json.tool | grep -E '"server_name"|"jetstream"|"cluster"' | head
```

### Pub/Sub simples com nats CLI

```bash
# nats CLI (ou use qualquer SDK go/python/js/java)
docker run --rm --network nats_default natsio/nats-box:0.14.5 \
  nats -s nats://nats1:4222 sub 'orders.>' &

docker run --rm --network nats_default natsio/nats-box:0.14.5 \
  nats -s nats://nats1:4222 pub orders.created '{"id":1,"amount":99.9}'
# subscriber acima deve receber
```

### JetStream (persistência)

```bash
# Cria stream
docker run --rm --network nats_default natsio/nats-box:0.14.5 \
  nats -s nats://nats1:4222 stream add ORDERS \
    --subjects="orders.>" --storage=file --retention=limits \
    --max-msgs=-1 --max-bytes=-1 --max-age=24h \
    --discard=old --replicas=3 --defaults

# Publica
docker run --rm --network nats_default natsio/nats-box:0.14.5 \
  nats -s nats://nats1:4222 pub orders.paid '{"id":42}'

# Consome com ack (at-least-once)
docker run --rm --network nats_default natsio/nats-box:0.14.5 \
  nats -s nats://nats1:4222 consumer add ORDERS worker --pull --defaults

# Info do stream (replicado em 3 nós)
docker run --rm --network nats_default natsio/nats-box:0.14.5 \
  nats -s nats://nats1:4222 stream info ORDERS
```

### Cleanup

```bash
docker compose down -v
```

### Tips de SRE

- **JetStream replica=3** em prod. Com replica=1 um disco queima, perdeu o stream.
- **NATS monitor** em `:8222` — scrape com Prometheus exporter oficial.
- **Subject naming**: use hierarquia (`events.payment.success`, `events.payment.failed`). Facilita wildcard sub (`events.payment.>`).
- **Back-pressure**: Core NATS descarta se subscriber é lento. Use JetStream + pull consumers para garantir entrega.
- **Leaf nodes** para clusters edge que não podem abrir porta. "Edge dials back" para o hub.
- `nats bench` mede latência/throughput — calibre antes de apontar para prod.

### References

- https://docs.nats.io/
- https://natsbyexample.com/
