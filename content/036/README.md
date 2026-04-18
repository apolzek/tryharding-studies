---
title: WebSocket vs Webhook — polyglot comparison (Go, Java, .NET, Node, Python)
tags: [networking, websocket, webhook, go, java, dotnet, nodejs, python, docker-compose, prometheus, k6, tls]
status: stable
---

# 036 — WebSocket vs Webhook

A didactic and complete PoC comparing **WebSocket** and **Webhook** implemented in **Go, Java, .NET, JavaScript (Node) and Python**, with individual Dockerfiles and `docker-compose` orchestrating everything. Includes a load producer, CLI client, browser client, Makefile, and this README as a study guide.

---

## Table of Contents

1. [TL;DR — when to use each](#1-tldr--when-to-use-each)
2. [Network protocol fundamentals](#2-network-protocol-fundamentals)
3. [Side-by-side comparison](#3-side-by-side-comparison)
4. [Common use cases (and anti-cases)](#4-common-use-cases-and-anti-cases)
5. [Project structure](#5-project-structure)
6. [Allocated ports](#6-allocated-ports)
7. [How to run — Docker](#7-how-to-run--docker)
8. [How to run — directly on the machine](#8-how-to-run--directly-on-the-machine)
9. [Testing manually](#9-testing-manually)
10. [Load and benchmarks](#10-load-and-benchmarks)
11. [Main challenges of each](#11-main-challenges-of-each)
12. [Monitoring](#12-monitoring)
13. [Troubleshooting](#13-troubleshooting)
14. [Behavior under high load](#14-behavior-under-high-load)
15. [Security](#15-security)
16. [Recommended readings](#16-recommended-readings)

---

## 1. TL;DR — when to use each

| You want...                                                          | Use        |
|----------------------------------------------------------------------|------------|
| Real-time updates, low latency, bidirectional                        | WebSocket  |
| Notify another system when an event happens (server-to-server)       | Webhook    |
| Chat, collaborative apps, trading, games, live telemetry             | WebSocket  |
| Integrate Stripe, GitHub, Slack, SaaS in general                     | Webhook    |
| Persistent connection with thousands/millions of clients in browser  | WebSocket  |
| Eventual delivery with retries, asynchronous processing              | Webhook    |

**Rule of thumb:** if the consumer is a human in a browser/app with an open session, think WebSocket. If the consumer is a backend service that wants to be "notified" about something, think Webhook.

---

## 2. Network protocol fundamentals

### 2.1 Webhook

- **Protocol:** HTTP/1.1 or HTTP/2 (regular request-response).
- **Direction:** unidirectional — whoever has the event does a `POST` on the receiver's endpoint.
- **Stateful?** No. Each request is independent.
- **Transport:** TCP (usually over TLS in production).
- **Payload:** typically JSON. Authenticity is guaranteed by **HMAC** (header `X-Signature`, `X-Hub-Signature-256` on GitHub, `Stripe-Signature`, etc).
- **Flow:**
  ```mermaid
  sequenceDiagram
      participant P as Producer (e.g. Stripe)
      participant R as Receiver (your service)
      P->>R: POST /webhook { event, data }
      Note over R: processes
      R-->>P: 2xx ok | 4xx drop | 5xx retry
  ```

### 2.2 WebSocket

- **Protocol:** RFC 6455. Starts as HTTP/1.1 with `Upgrade: websocket` + `Connection: Upgrade` and becomes a full-duplex protocol over the same TCP connection.
- **Handshake:**
  ```
  GET /ws HTTP/1.1
  Host: example.com
  Upgrade: websocket
  Connection: Upgrade
  Sec-WebSocket-Key: <base64>
  Sec-WebSocket-Version: 13
  ```
  Response: `101 Switching Protocols`.
- **Direction:** bidirectional and full-duplex. Either side can write at any moment.
- **Stateful?** Yes. Connection stays open; the server keeps per-client state.
- **Transport:** the same TCP connection from the HTTP handshake. URLs are `ws://` (TCP:80) or `wss://` (TLS:443).
- **Frames:** messages are split into frames (text, binary, ping, pong, close). There is flow control and payload masking (client→server always masked for safety against proxies).
- **Keep-alive:** ping/pong every 20–30s is standard.

### 2.3 The "problem" each one solves

HTTP was designed pull-based: the client asks, the server responds. For "event push" we have three historical approaches:

1. **Long polling** — client sends a request and the server holds it until data is available. Ugly, but compatible.
2. **Server-Sent Events (SSE)** — unidirectional server→client stream over HTTP. Simple, but one-way only.
3. **WebSocket** — real full-duplex.

Webhook is the other direction: the **event server** acts as an HTTP client and talks to an endpoint of yours. It's not a "new protocol", it's just HTTP used with convention.

---

## 3. Side-by-side comparison

| Aspect                   | Webhook                                      | WebSocket                                       |
|--------------------------|----------------------------------------------|-------------------------------------------------|
| Base protocol            | HTTP (1.1/2)                                 | HTTP → upgrade → RFC 6455                       |
| Connection               | Short (one per event)                        | Long, persistent                                |
| Direction                | 1-way (producer → receiver)                  | Full-duplex                                     |
| Typical latency          | Tens to hundreds of ms                       | Milliseconds                                    |
| Server-side state        | None                                         | Per client                                      |
| Horizontal scale         | Trivial (stateless, behind LB)               | Requires *sticky sessions* or pub/sub           |
| Guaranteed delivery      | Via producer retry                           | Not part of protocol — app needs ACK            |
| Firewalls/NAT            | Any HTTP client goes through                 | Same (starts as HTTP), but long-lived           |
| Idle cost                | Zero (no connection)                         | Memory/FD per open connection                   |
| Browser-friendly?        | Doesn't make sense for browser               | Native (`new WebSocket(...)`)                   |
| Observability            | Easy (normal HTTP logs)                      | Harder (persistent connection)                  |
| Testability              | `curl`, `httpie`                             | `wscat`, `websocat`, browser                    |

---

## 4. Common use cases (and anti-cases)

### Webhook — where it shines

- **SaaS integrations**: Stripe notifies payment confirmed, GitHub notifies push/PR, Slack forwards slash-command.
- **Event-driven between services**: one service notifies `N` others when something happens (usually via a webhook gateway / bus).
- **CI/CD**: GitHub triggers webhook → Jenkins/Actions starts build.
- **Billing/audit**: important events that can be processed asynchronously with retry.

### Webhook — where it doesn't make sense

- **Real-time UI updates**: the browser doesn't listen to webhooks.
- **Bidirectional flows** (client input that the server needs to process continuously).
- **Latency < 50ms** perceived by the end user — each webhook carries TLS/TCP setup overhead.

### WebSocket — where it shines

- **Chat and collaboration** (Slack, Google Docs, Figma).
- **Trading / quotes / live dashboards**.
- **Multiplayer games** (matchmaking is REST; gameplay is WS or UDP).
- **IoT** where the device needs to receive commands in addition to sending telemetry.
- **Real-time notifications** in the browser (alternative to SSE when you also need to send things from the client).

### WebSocket — where it doesn't make sense

- **One-off server-to-server integration**: webhook/HTTP request is simpler and stateless.
- **Rare events** (1 per hour): not worth the open connection.
- **Clients behind aggressive proxies** that close long connections (use fallback via long-polling — libraries like Socket.IO do this).

---

## 5. Project structure

```
036/
├── README.md                  # this file
├── Makefile                   # shortcuts (build/up/down/tests)
├── docker-compose.yml         # orchestrates 10 services + producer
│
├── websocket/
│   ├── go/         server.go        (gorilla/websocket)
│   ├── java/       WsServer.java    (Javalin)
│   ├── dotnet/     Program.cs       (ASP.NET Core, System.Net.WebSockets)
│   ├── javascript/ server.js        (ws)
│   └── python/     server.py        (websockets)
│
├── webhook/
│   ├── go/         server.go        (net/http)
│   ├── java/       WhServer.java    (Javalin)
│   ├── dotnet/     Program.cs       (Minimal API)
│   ├── javascript/ server.js        (native http)
│   └── python/     server.py        (FastAPI + uvicorn)
│
├── producer/
│   ├── producer.py            # generates load on the webhooks and WS
│   └── Dockerfile
│
└── clients/
    ├── ws_client.py           # Python CLI client
    └── index.html             # browser client (open directly in the browser)
```

Each implementation has:
- Minimal server code, idiomatic to the language.
- Multi-stage `Dockerfile` (build → lean runtime).
- Health check at `/health` (webhooks) and counter at `/stats`.

---

## 6. Allocated ports

| Service           | WebSocket | Webhook |
|-------------------|-----------|---------|
| Go                | 8001      | 9001    |
| Java              | 8002      | 9002    |
| .NET              | 8003      | 9003    |
| JavaScript (Node) | 8004      | 9004    |
| Python            | 8005      | 9005    |

Recipe: `80XX` is WS, `90XX` is webhook. Last digit is the language.

---

## 7. How to run — Docker

```bash
# build all 10 images
make build

# bring everything up
make up
make ps

# aggregated logs (Ctrl-C exits, services keep running)
make logs

# tear down
make down
```

> The first `build` of Java/.NET takes a while (Maven resolves deps, dotnet restore). After that, cache helps.

---

## 8. How to run — directly on the machine

Useful for debugging with breakpoints and local tools.

**Go:**
```bash
make local-ws-go    # :8001
make local-wh-go    # :9001
```

**Python:**
```bash
make local-ws-py    # :8005
make local-wh-py    # :9005
```

**Node:**
```bash
make local-ws-js    # :8004
make local-wh-js    # :9004
```

**Java/.NET** (no specific target because it requires a toolchain):
```bash
cd websocket/java && mvn -q package && java -jar target/ws-server-jar-with-dependencies.jar
cd websocket/dotnet && dotnet run
```

---

## 9. Testing manually

### 9.1 Webhook — `curl`

```bash
BODY='{"event":"order.created","id":42,"amount":99}'
SIG=$(printf '%s' "$BODY" | openssl dgst -sha256 -hmac s3cret | awk '{print $2}')

curl -sS -X POST http://localhost:9001/webhook \
  -H "Content-Type: application/json" \
  -H "X-Signature: $SIG" \
  -d "$BODY"
# {"status":"accepted"}
```

Fire in a single command against all languages:
```bash
make curl-webhook
```

See how many each one received:
```bash
make stats
# --- :9001  {"received":3,"ts":1745...}
# --- :9002  {"received":3,"ts":1745...}
# ...
```

### 9.2 WebSocket — `wscat`

```bash
npm i -g wscat
wscat -c ws://localhost:8001/ws
> hello
< hello          # the server broadcasts it back
```

### 9.3 WebSocket — browser

Open `clients/index.html` in your browser, pick the server and connect. It lets you inspect frames in DevTools → Network → WS.

### 9.4 WebSocket — interactive Python client

```bash
pip install websockets
python clients/ws_client.py ws://localhost:8005
```

---

## 10. Load and benchmarks

The `producer` container sends events in parallel to the 5 webhooks:

```bash
# 50 req/s/target for 10s
make burst

# output
# [producer] sent=250 failed=0 avg_latency=3.2ms
# ...

# check who received how much
make stats
```

Publish a message to each WS simultaneously (to see broadcast):
```bash
make ws-notify
```

For more serious tests use:
- **Webhook:** `wrk`, `hey`, `vegeta`, `k6`.
- **WebSocket:** `artillery`, `k6 ws`, `tsung`, `thor`.

`k6` example:
```js
import ws from 'k6/ws';
export const options = { vus: 1000, duration: '30s' };
export default function () {
  ws.connect('ws://localhost:8001/ws', {}, (socket) => {
    socket.on('open', () => socket.send('hi'));
    socket.setTimeout(() => socket.close(), 5000);
  });
}
```

---

## 11. Main challenges of each

### 11.1 Webhook

| Challenge               | Discussion                                                                                                                               |
|-------------------------|------------------------------------------------------------------------------------------------------------------------------------------|
| **Idempotency**         | The producer will retry — the receiver needs to deduplicate by `event_id`.                                                               |
| **Retry / back-pressure** | If the receiver responds slowly or 5xx, the producer fills the queue. Configure exponential backoff and DLQ.                           |
| **Origin verification** | Any IP can POST to your endpoint. Use HMAC (`X-Signature`), IP allow-list, or mTLS.                                                      |
| **Replay attack**       | Include a timestamp in the signed payload and reject events that are too old.                                                            |
| **Ordering**            | Webhooks generally **do not** guarantee order. Design to be order-independent or use `sequence_id`.                                     |
| **Visibility**          | Debugging is "look at the log": use [`smee.io`](https://smee.io), `webhook.site`, or ngrok to inspect.                                   |
| **Silent loss**         | If the receiver was down during the producer's retries, the event is lost. Always have a "resync" / event-listing endpoint.              |

### 11.2 WebSocket

| Challenge               | Discussion                                                                                                                               |
|-------------------------|------------------------------------------------------------------------------------------------------------------------------------------|
| **Horizontal scaling**  | Connection is stateful. You need sticky sessions (L4 LB or hash by IP/cookie) and pub/sub (Redis, NATS) for cross-pod broadcast.         |
| **File descriptors**    | Each client = 1 FD. Tune `ulimit -n`, `somaxconn`, `net.core.somaxconn`.                                                                 |
| **Keep-alive / zombies**| TCP connections can stay "half-closed" for hours. Implement app-level ping/pong and timeout.                                             |
| **Back-pressure**       | A slow client causes the server's write buffer to grow endlessly. Monitor per-connection queue size and drop slow clients.               |
| **Reconnection**        | Network drops are normal. The client needs to reconnect with exponential backoff and resume state (request snapshot + replay diff).      |
| **Ordering / reliability** | Messages in flight at disconnect time are lost. Use `message_id` + ACK if you need at-least-once.                                     |
| **Proxies / CDN**       | Not every HTTP proxy preserves `Upgrade`. CloudFlare/AWS ALB require specific config. Idle timeouts may close connections silently.      |
| **Fanout**              | In-memory broadcast is simple for 1 pod. With multiple pods, you need a bus (Redis Pub/Sub, Kafka, NATS).                                |
| **Security**            | No standard WS equivalent of CORS — use `Origin` check, tokens in the first frame, always TLS.                                           |

---

## 12. Monitoring

### 12.1 Webhook — essential metrics

- **Producer:**
  - `webhook_send_total{dest,event_type,status}` — counter (label per HTTP status class).
  - `webhook_send_duration_seconds` — latency histogram.
  - `webhook_queue_depth` — how many events waiting for retry.
  - `webhook_retries_total{dest}`.
  - `webhook_dlq_total` — events that went to dead-letter.
- **Receiver:**
  - `webhook_received_total{event_type}`.
  - `webhook_processing_duration_seconds`.
  - `webhook_invalid_signature_total` — attack / mis-configured producer.
  - `webhook_duplicates_total` — after dedup.

### 12.2 WebSocket — essential metrics

- `ws_connections_active` — **gauge**, connections open right now.
- `ws_connections_total{reason="closed"|"error"}` — cumulative counter.
- `ws_connection_duration_seconds` — histogram (how long each connection lives).
- `ws_messages_received_total{direction="in|out"}`.
- `ws_message_bytes` — size histogram.
- `ws_send_queue_length` — per-connection back-pressure (p99).
- `ws_ping_rtt_seconds`.

### 12.3 Tooling

- **Prometheus + Grafana** for all of the above.
- **OpenTelemetry** for tracing — webhook creates a new trace; WS can propagate `traceparent` on the first frame/handshake headers.
- **Structured logs**: always log `event_id`, `conn_id`, `client_ip`.
- **Key dashboards:**
  - Webhook delivery rate vs status code (stacked).
  - Active WS connections per pod + p99 of send-queue.

---

## 13. Troubleshooting

### 13.1 Webhook

**Symptom: "My endpoint isn't receiving anything"**
1. On the producer side: is it actually firing? (its log / panel)
2. Is the URL reachable from the internet? (`curl` from elsewhere)
3. Does your LB/firewall allow POST? (not every WAF does)
4. Does DNS resolve? Valid certificate? `openssl s_client -connect host:443`.

**Symptom: "I receive it, but the producer says it failed"**
- You responded `2xx` in **less** than the producer's timeout (usually 5–10s).
- Process asynchronously: reply 202 quickly, enqueue, process later.

**Symptom: "I'm receiving the same event 5 times"**
- Normal. You took too long or responded 5xx. Dedup by `event_id`.

**Symptom: "Invalid signature"**
- Check the **exact** body encoding (don't re-encode the JSON!). Some libs re-serialize and break HMAC.
- Check the correct algorithm (SHA-256 is standard; some use base64 instead of hex).

**Tools:**
```bash
# receive a local webhook from the internet
ngrok http 9001
# capture and inspect
https://webhook.site
# event replay (Stripe CLI)
stripe trigger payment_intent.succeeded
```

### 13.2 WebSocket

**Symptom: "Connects and drops in 60s"**
- Some proxy/LB with `idle_timeout`. Tune the LB (AWS ALB: `Idle Timeout` → 3600s; CloudFlare: enable WS; nginx: `proxy_read_timeout`).
- Implement ping/pong more frequently than the timeout.

**Symptom: "Handshake 400 / Upgrade failed"**
- Missing `Upgrade: websocket` / `Connection: Upgrade` header. Proxy drop.
- Wrong `Sec-WebSocket-Version`.
- Strict origin check on the server rejected it. See the `Access-Control-Allow-Origin` equivalent (app-level).

**Symptom: "Messages arrive with huge delay under load"**
- Back-pressure: a slow client is stalling the writer loop. Implement a bounded queue + drop policy.
- TCP Nagle: disable with `TCP_NODELAY`.

**Symptom: "Memory grows until OOM"**
- Unbounded write buffer per connection.
- Close dead sessions (half-open): ping/pong with timeout; read with deadline.

**Tools:**
```bash
# inspect handshake and frames
wscat -c ws://localhost:8001/ws
websocat ws://localhost:8001/ws
# server side: how many connections
ss -tna state established | grep :8001 | wc -l
# packet capture (handshake is HTTP, upgrade is text)
tcpdump -i any -A -s 0 port 8001
```

Chrome DevTools → Network → filter **WS** → click the connection → **Messages** tab shows all the frames.

---

## 14. Behavior under high load

### 14.1 Webhook

- **Usual bottleneck:** the receiver. If the producer sends 10k/s and the receiver processes synchronously, it fills up.
- **Correct pattern:** `POST → validation + enqueue to broker → 202` quickly. Worker consumes the queue.
- **Retry storms:** when the receiver goes down, the producer retries exponentially — but if there are 1000 independent producers, the thundering herd brings everything down again when it comes back up. Solution: jitter in the backoff, circuit breaker on the producer.
- **Horizontal scaling:** trivial. L7 LB in front, N stateless replicas.
- **Marginal cost:** low per event, but every new TCP connection has an expensive handshake. Use connection pooling on the producer (`keep-alive`).

### 14.2 WebSocket

Reference numbers (modern Linux, decent server):
- 1 Node/Go process can hold ~**50k–200k idle connections** with good tuning (ulimit, kernel).
- With real traffic (messages, broadcast, logic), it drops to ~10k–50k per instance.

Critical care:
- **Cross-pod fanout:** with 10 pods and 100k clients each, a broadcast message needs to reach 1M. Without a bus, each pod only knows its own clients. Use **Redis Pub/Sub** (simple) or **NATS/Kafka** (robust).
- **Sticky sessions:** HTTP cookies or IP hash on the LB. Otherwise, each message from the same client may land on a pod that doesn't know the session.
- **Connection storms:** when a pod restarts, all clients reconnect at the same time. Jitter on the client + *connection draining* on the server.
- **Memory per conn:** in Node.js with `ws`, ~10-20KB per idle connection; in Go, ~4-8KB; depends on the configured write-buffer size.

### 14.3 Choosing between them for "push"

| Scenario                                       | Choice                                                      |
|------------------------------------------------|-------------------------------------------------------------|
| 100k events/s backend→backend                  | Webhook (or better: Kafka/NATS, webhook for external integration) |
| 100k clients receiving near-live updates       | WebSocket with Redis pub/sub                                |
| 10 events/day, needs to be reliable            | Webhook with retries                                        |
| Client that **responds** to the server in real time | WebSocket (only it is bidirectional)                   |

---

## 15. Security

### Webhook

1. **HMAC SHA-256 on the body** with a shared secret. Always in constant time (`hmac.compare_digest`).
2. **Timestamp in the payload** + window (e.g. reject > 5min) to avoid replay.
3. **IP allow-list** when the producer has stable IPs (Stripe, GitHub publish ranges).
4. **Rate limit** on the endpoint.
5. **Mandatory TLS** (and `HSTS` at the origin).

### WebSocket

1. **Always TLS (`wss://`)** in production — otherwise proxies can inject.
2. **Authentication on the handshake** via `Authorization: Bearer`, session cookie, or token in the query-string (less secure, ends up in logs).
3. **`Origin` check** server-side — analogous to fetch CORS.
4. **Rate-limit per IP/user** on accept and on messages.
5. **Input validation** on every frame received — parser must be robust to malformed JSON.
6. **Don't expose PII in broadcast** without per-user filtering.

---

## 16. Recommended readings

- **Webhook**
  - Stripe docs — "Receiving webhooks" (one of the best real-world guides).
  - "webhooks.fyi" — catalog of practices across SaaS.
  - RFC 8935 — "Server-to-Server Event Notifications" (still rare, but relevant).
- **WebSocket**
  - RFC 6455 — "The WebSocket Protocol".
  - MDN — "Writing WebSocket servers".
  - High-scale case studies: Slack, Discord and Figma publish good blog posts about WS scaling.

---

## Appendix — quick-test cheatsheet

```bash
# 1. Bring everything up
make build && make up

# 2. Fire a webhook to each language
make curl-webhook

# 3. See counters
make stats

# 4. Open a WS client
wscat -c ws://localhost:8001/ws

# 5. 50 req/s load for 10s on all webhooks
make burst

# 6. A test broadcast on each WS
make ws-notify

# 7. Tear down
make down
```

Happy exploring. 🐙
