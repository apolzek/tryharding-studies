# 036 — WebSocket vs Webhook

PoC didático e completo comparando **WebSocket** e **Webhook** implementados em **Go, Java, .NET, JavaScript (Node) e Python**, com Dockerfiles individuais e `docker-compose` orquestrando tudo. Inclui producer de carga, cliente CLI, cliente browser, Makefile e este README como guia de estudo.

---

## Sumário

1. [TL;DR — quando usar cada um](#1-tldr--quando-usar-cada-um)
2. [Fundamentos de protocolo de rede](#2-fundamentos-de-protocolo-de-rede)
3. [Comparativo lado a lado](#3-comparativo-lado-a-lado)
4. [Casos de uso comuns (e anti-casos)](#4-casos-de-uso-comuns-e-anti-casos)
5. [Estrutura do projeto](#5-estrutura-do-projeto)
6. [Portas alocadas](#6-portas-alocadas)
7. [Como rodar — Docker](#7-como-rodar--docker)
8. [Como rodar — direto na máquina](#8-como-rodar--direto-na-máquina)
9. [Testando manualmente](#9-testando-manualmente)
10. [Carga e benchmarks](#10-carga-e-benchmarks)
11. [Principais desafios de cada um](#11-principais-desafios-de-cada-um)
12. [Monitoramento](#12-monitoramento)
13. [Troubleshooting](#13-troubleshooting)
14. [Comportamento sob alta carga](#14-comportamento-sob-alta-carga)
15. [Segurança](#15-segurança)
16. [Leituras recomendadas](#16-leituras-recomendadas)

---

## 1. TL;DR — quando usar cada um

| Você quer...                                                         | Use        |
|----------------------------------------------------------------------|------------|
| Atualizações em tempo real, baixa latência, bidirecional             | WebSocket  |
| Notificar outro sistema quando um evento acontece (server-to-server) | Webhook    |
| Chat, colaborativo, trading, jogos, telemetria ao vivo               | WebSocket  |
| Integrar Stripe, GitHub, Slack, SaaS em geral                        | Webhook    |
| Conexão persistente com milhares/milhões de clientes no browser      | WebSocket  |
| Entrega eventual com retries, processamento assíncrono               | Webhook    |

**Regra prática:** se o consumidor é humano num browser/app com sessão aberta, pense WebSocket. Se o consumidor é um serviço backend que quer ser "avisado" sobre algo, pense Webhook.

---

## 2. Fundamentos de protocolo de rede

### 2.1 Webhook

- **Protocolo:** HTTP/1.1 ou HTTP/2 (request-response normal).
- **Direção:** unidirecional — quem tem o evento faz `POST` no endpoint do receptor.
- **Stateful?** Não. Cada requisição é independente.
- **Transport:** TCP (geralmente sobre TLS em produção).
- **Payload:** tipicamente JSON. A autenticidade é garantida por **HMAC** (header `X-Signature`, `X-Hub-Signature-256` no GitHub, `Stripe-Signature`, etc).
- **Fluxo:**
  ```mermaid
  sequenceDiagram
      participant P as Produtor (ex.: Stripe)
      participant R as Receptor (seu serviço)
      P->>R: POST /webhook { event, data }
      Note over R: processa
      R-->>P: 2xx ok | 4xx drop | 5xx retry
  ```

### 2.2 WebSocket

- **Protocolo:** RFC 6455. Começa como HTTP/1.1 com `Upgrade: websocket` + `Connection: Upgrade` e passa a ser um protocolo full-duplex sobre a mesma conexão TCP.
- **Handshake:**
  ```
  GET /ws HTTP/1.1
  Host: example.com
  Upgrade: websocket
  Connection: Upgrade
  Sec-WebSocket-Key: <base64>
  Sec-WebSocket-Version: 13
  ```
  Resposta: `101 Switching Protocols`.
- **Direção:** bidirecional e full-duplex. Qualquer lado pode escrever a qualquer momento.
- **Stateful?** Sim. Conexão fica aberta; o servidor mantém estado por cliente.
- **Transport:** mesma conexão TCP do handshake HTTP. URLs são `ws://` (TCP:80) ou `wss://` (TLS:443).
- **Frames:** mensagens são divididas em frames (text, binary, ping, pong, close). Há controle de fluxo e payload masking (client→server sempre mascarado por segurança contra proxies).
- **Keep-alive:** ping/pong a cada 20–30s é padrão.

### 2.3 O "problema" que cada um resolve

HTTP foi desenhado pull-based: o cliente pergunta, o servidor responde. Para "evento push" temos três abordagens históricas:

1. **Long polling** — cliente faz request e o servidor segura até ter dado. Feio, mas compatível.
2. **Server-Sent Events (SSE)** — stream unidirecional servidor→cliente sobre HTTP. Simples, mas só um sentido.
3. **WebSocket** — full-duplex real.

Webhook é a outra direção: o **servidor de eventos** age como cliente HTTP e fala com um endpoint seu. Não é um "protocolo novo", é só HTTP usado com convenção.

---

## 3. Comparativo lado a lado

| Aspecto                  | Webhook                                      | WebSocket                                       |
|--------------------------|----------------------------------------------|-------------------------------------------------|
| Protocolo base           | HTTP (1.1/2)                                 | HTTP → upgrade → RFC 6455                       |
| Conexão                  | Curta (uma por evento)                       | Longa, persistente                              |
| Direção                  | 1-way (produtor → receptor)                  | Full-duplex                                     |
| Latência típica          | Dezenas a centenas de ms                     | Milissegundos                                   |
| Estado no servidor       | Nenhum                                       | Por cliente                                     |
| Escala horizontal        | Trivial (stateless, atrás de LB)             | Exige *sticky sessions* ou pub/sub              |
| Entrega garantida        | Por retry do produtor                        | Não é do protocolo — app precisa ACK            |
| Firewalls/NAT            | Qualquer cliente HTTP atravessa              | Idem (começa como HTTP), mas long-lived         |
| Custo de idle            | Zero (sem conexão)                           | Memória/FD por conexão aberta                   |
| Browser-friendly?        | Não faz sentido para browser                 | Nativo (`new WebSocket(...)`)                   |
| Observabilidade          | Fácil (logs HTTP normais)                    | Mais difícil (conexão persistente)              |
| Testabilidade            | `curl`, `httpie`                             | `wscat`, `websocat`, navegador                  |

---

## 4. Casos de uso comuns (e anti-casos)

### Webhook — onde brilha

- **Integrações SaaS**: Stripe avisa pagamento confirmado, GitHub avisa push/PR, Slack encaminha slash-command.
- **Event-driven entre serviços**: um serviço notifica `N` outros quando algo acontece (geralmente via um gateway de webhook / bus).
- **CI/CD**: GitHub dispara webhook → Jenkins/Actions inicia build.
- **Billing/audit**: eventos importantes que podem ser processados assíncronamente com retry.

### Webhook — onde não faz sentido

- **Atualizações de UI em tempo real**: o browser não escuta webhooks.
- **Fluxos bidirecionais** (input do cliente que o servidor precisa processar continuamente).
- **Latência < 50ms** percebida pelo usuário final — cada webhook carrega overhead de TLS/TCP setup.

### WebSocket — onde brilha

- **Chat e colaboração** (Slack, Google Docs, Figma).
- **Trading / cotações / dashboards ao vivo**.
- **Jogos multiplayer** (matchmaking é REST; gameplay é WS ou UDP).
- **IoT** onde o dispositivo precisa receber comandos além de enviar telemetria.
- **Notificações em tempo real** no browser (alternativa a SSE quando você também precisa enviar coisas do cliente).

### WebSocket — onde não faz sentido

- **Integração server-to-server pontual**: webhook/HTTP request é mais simples e stateless.
- **Eventos raros** (1 por hora): não vale a conexão aberta.
- **Clientes atrás de proxies agressivos** que fecham conexões longas (use fallback via long-polling — bibliotecas como Socket.IO fazem isso).

---

## 5. Estrutura do projeto

```
036/
├── README.md                  # este arquivo
├── Makefile                   # atalhos (build/up/down/tests)
├── docker-compose.yml         # orquestra 10 serviços + producer
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
│   ├── javascript/ server.js        (http nativo)
│   └── python/     server.py        (FastAPI + uvicorn)
│
├── producer/
│   ├── producer.py            # gera carga nos webhooks e WS
│   └── Dockerfile
│
└── clients/
    ├── ws_client.py           # cliente CLI Python
    └── index.html             # cliente browser (abra direto no navegador)
```

Cada implementação tem:
- Código de servidor mínimo, idiomático da linguagem.
- `Dockerfile` multi-stage (build → runtime enxuto).
- Health check em `/health` (webhooks) e contador em `/stats`.

---

## 6. Portas alocadas

| Serviço           | WebSocket | Webhook |
|-------------------|-----------|---------|
| Go                | 8001      | 9001    |
| Java              | 8002      | 9002    |
| .NET              | 8003      | 9003    |
| JavaScript (Node) | 8004      | 9004    |
| Python            | 8005      | 9005    |

Receita: `80XX` é WS, `90XX` é webhook. Último dígito é a linguagem.

---

## 7. Como rodar — Docker

```bash
# build de todas as 10 imagens
make build

# sobe tudo
make up
make ps

# logs agregados (Ctrl-C sai, serviços continuam)
make logs

# derruba
make down
```

> Primeira `build` do Java/.NET demora (Maven resolve deps, dotnet restore). Depois cache ajuda.

---

## 8. Como rodar — direto na máquina

Útil para debugar com breakpoints e ferramentas locais.

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

**Java/.NET** (sem target específico porque exige toolchain):
```bash
cd websocket/java && mvn -q package && java -jar target/ws-server-jar-with-dependencies.jar
cd websocket/dotnet && dotnet run
```

---

## 9. Testando manualmente

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

Dispara no mesmo comando para todas as linguagens:
```bash
make curl-webhook
```

Ver quantos cada um recebeu:
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
< hello          # o servidor faz broadcast de volta
```

### 9.3 WebSocket — browser

Abra `clients/index.html` no navegador, escolha o servidor e conecte. Permite inspecionar frames no DevTools → Network → WS.

### 9.4 WebSocket — cliente Python interativo

```bash
pip install websockets
python clients/ws_client.py ws://localhost:8005
```

---

## 10. Carga e benchmarks

O container `producer` envia eventos em paralelo para os 5 webhooks:

```bash
# 50 req/s/alvo durante 10s
make burst

# saída
# [producer] sent=250 failed=0 avg_latency=3.2ms
# ...

# confere quem recebeu quanto
make stats
```

Publicar uma mensagem em cada WS simultaneamente (para ver broadcast):
```bash
make ws-notify
```

Para testes mais sérios use:
- **Webhook:** `wrk`, `hey`, `vegeta`, `k6`.
- **WebSocket:** `artillery`, `k6 ws`, `tsung`, `thor`.

Exemplo `k6`:
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

## 11. Principais desafios de cada um

### 11.1 Webhook

| Desafio                 | Discussão                                                                                                                                |
|-------------------------|------------------------------------------------------------------------------------------------------------------------------------------|
| **Idempotência**        | O produtor fará retry — o receptor precisa deduplicar por `event_id`.                                                                    |
| **Retry / back-pressure** | Se o receptor responde lento ou 5xx, o produtor enche fila. Configure backoff exponencial e DLQ.                                       |
| **Verificação de origem** | Qualquer IP pode fazer POST no seu endpoint. Use HMAC (`X-Signature`), IP allow-list ou mTLS.                                          |
| **Replay attack**       | Inclua timestamp no payload assinado e rejeite eventos muito antigos.                                                                    |
| **Ordering**            | Webhooks normalmente **não** garantem ordem. Projete para ser order-independent ou use `sequence_id`.                                   |
| **Visibilidade**        | Debugar é "olha pro log": use [`smee.io`](https://smee.io), `webhook.site`, ou ngrok para inspecionar.                                   |
| **Perda silenciosa**    | Se o receptor estava down durante retries do produtor, evento perdido. Sempre ter endpoint de "resync" / listagem de eventos.            |

### 11.2 WebSocket

| Desafio                 | Discussão                                                                                                                                |
|-------------------------|------------------------------------------------------------------------------------------------------------------------------------------|
| **Escala horizontal**   | Conexão é stateful. Você precisa de sticky sessions (LB L4 ou hash por IP/cookie) e pub/sub (Redis, NATS) para broadcast cross-pod.      |
| **File descriptors**    | Cada cliente = 1 FD. Ajuste `ulimit -n`, `somaxconn`, `net.core.somaxconn`.                                                              |
| **Keep-alive / zombies**| Conexões TCP podem ficar "meio-fechadas" por horas. Implemente ping/pong app-level e timeout.                                            |
| **Back-pressure**       | Cliente lento faz buffer de escrita do servidor crescer sem fim. Monitore tamanho da fila por conexão e derrube clientes lentos.         |
| **Reconexão**           | Queda de rede é normal. Cliente precisa reconectar com backoff exponencial e retomar estado (pedir snapshot + replay diff).              |
| **Ordering / reliability** | Mensagens em voo no momento do disconnect se perdem. Use `message_id` + ACK se precisar de at-least-once.                             |
| **Proxies / CDN**       | Nem todo proxy HTTP preserva `Upgrade`. CloudFlare/AWS ALB exigem config específica. Timeouts de idle podem fechar conexão sem aviso.    |
| **Fanout**              | Broadcast em memória é simples para 1 pod. Com vários pods, precisa barramento (Redis Pub/Sub, Kafka, NATS).                             |
| **Segurança**           | Sem CORS equivalente no WS padrão — use `Origin` check, tokens no primeiro frame, TLS sempre.                                            |

---

## 12. Monitoramento

### 12.1 Webhook — métricas essenciais

- **Produtor:**
  - `webhook_send_total{dest,event_type,status}` — contador (label por HTTP status class).
  - `webhook_send_duration_seconds` — histogram de latência.
  - `webhook_queue_depth` — quantos eventos esperando retry.
  - `webhook_retries_total{dest}`.
  - `webhook_dlq_total` — eventos que foram pra dead-letter.
- **Receptor:**
  - `webhook_received_total{event_type}`.
  - `webhook_processing_duration_seconds`.
  - `webhook_invalid_signature_total` — ataque / produtor mal-configurado.
  - `webhook_duplicates_total` — depois do dedup.

### 12.2 WebSocket — métricas essenciais

- `ws_connections_active` — **gauge**, conexões abertas agora.
- `ws_connections_total{reason="closed"|"error"}` — contador cumulativo.
- `ws_connection_duration_seconds` — histograma (quanto tempo cada conexão vive).
- `ws_messages_received_total{direction="in|out"}`.
- `ws_message_bytes` — histograma de tamanho.
- `ws_send_queue_length` — back-pressure por conexão (p99).
- `ws_ping_rtt_seconds`.

### 12.3 Ferramental

- **Prometheus + Grafana** para tudo acima.
- **OpenTelemetry** para tracing — webhook cria novo trace; WS pode propagar `traceparent` no primeiro frame/headers do handshake.
- **Logs estruturados**: sempre logar `event_id`, `conn_id`, `client_ip`.
- **Dashboards chave:**
  - Taxa de entrega do webhook vs status code (stacked).
  - Conexões WS ativas por pod + p99 do send-queue.

---

## 13. Troubleshooting

### 13.1 Webhook

**Sintoma: "Meu endpoint não recebe nada"**
1. Do lado do produtor: ele realmente está disparando? (log dele / painel)
2. A URL é alcançável pela internet? (`curl` de outro lugar)
3. Seu LB/firewall passa POST? (nem todo WAF passa)
4. DNS resolve? Certificado válido? `openssl s_client -connect host:443`.

**Sintoma: "Recebo, mas o produtor diz que falhou"**
- Você respondeu `2xx` em **menos** que o timeout do produtor (geralmente 5–10s).
- Processe assíncrono: responde 202 rápido, coloca em fila, processa depois.

**Sintoma: "Estou recebendo o mesmo evento 5 vezes"**
- Normal. Você demorou demais ou respondeu 5xx. Dedup por `event_id`.

**Sintoma: "Assinatura inválida"**
- Confira encoding do body **exato** (sem recodificar JSON!). Algumas libs re-serializam e quebram HMAC.
- Checar o algoritmo correto (SHA-256 é o padrão; alguns usam base64 ao invés de hex).

**Ferramentas:**
```bash
# receber webhook local a partir da internet
ngrok http 9001
# capturar e inspecionar
https://webhook.site
# replay de eventos (Stripe CLI)
stripe trigger payment_intent.succeeded
```

### 13.2 WebSocket

**Sintoma: "Conecta e cai em 60s"**
- Algum proxy/LB com `idle_timeout`. Ajuste o LB (AWS ALB: `Idle Timeout` → 3600s; CloudFlare: habilite WS; nginx: `proxy_read_timeout`).
- Implemente ping/pong mais frequente que o timeout.

**Sintoma: "Handshake 400 / Upgrade failed"**
- Faltou header `Upgrade: websocket` / `Connection: Upgrade`. Proxy drop.
- `Sec-WebSocket-Version` errada.
- Origin check estrito no servidor rejeitou. Veja `Access-Control-Allow-Origin` equivalente (app-level).

**Sintoma: "Mensagens chegam com atraso enorme sob carga"**
- Back-pressure: cliente lento está segurando o writer loop. Implemente fila bounded + drop policy.
- TCP Nagle: desabilitar com `TCP_NODELAY`.

**Sintoma: "Memória cresce até OOM"**
- Buffer de escrita ilimitado por conexão.
- Fechar sessões mortas (half-open): ping/pong com timeout; ler com deadline.

**Ferramentas:**
```bash
# inspecionar handshake e frames
wscat -c ws://localhost:8001/ws
websocat ws://localhost:8001/ws
# lado servidor: quantas conexões
ss -tna state established | grep :8001 | wc -l
# captura pacote (handshake é HTTP, upgrade é texto)
tcpdump -i any -A -s 0 port 8001
```

Chrome DevTools → Network → filtro **WS** → clique na conexão → aba **Messages** mostra todos os frames.

---

## 14. Comportamento sob alta carga

### 14.1 Webhook

- **Gargalo usual:** o receptor. Se o produtor manda 10k/s e o receptor processa síncrono, enche.
- **Padrão correto:** `POST → validação + enqueue em broker → 202` rápido. Worker consome a fila.
- **Retry storms:** quando o receptor cai, o produtor retenta exponencialmente — mas se houver 1000 produtores independentes, o thundering herd derruba tudo de novo quando sobe. Solução: jitter no backoff, circuit breaker no produtor.
- **Horizontal scaling:** trivial. LB (L7) na frente, N réplicas stateless.
- **Custo marginal:** baixo por evento, mas cada conexão TCP nova tem handshake caro. Use connection pooling no produtor (`keep-alive`).

### 14.2 WebSocket

Números de referência (Linux moderno, servidor decente):
- 1 processo Node/Go consegue ~**50k–200k conexões ociosas** se tuning bom (ulimit, kernel).
- Com tráfego real (mensagens, broadcast, lógica), cai a ~10k–50k por instância.

Cuidados críticos:
- **Fanout cross-pod:** com 10 pods e 100k clientes cada, uma mensagem de broadcast precisa chegar aos 1M. Sem barramento, cada pod só conhece os próprios clientes. Use **Redis Pub/Sub** (simples) ou **NATS/Kafka** (robusto).
- **Sticky sessions:** HTTP cookies ou hash por IP no LB. Caso contrário, cada mensagem do mesmo cliente pode cair num pod que não conhece a sessão.
- **Connection storms:** ao restart de um pod, todos os clientes reconectam ao mesmo tempo. Jitter no cliente + *connection draining* no servidor.
- **Memory per conn:** em Node.js com `ws`, ~10-20KB por conexão idle; em Go, ~4-8KB; depende do tamanho do buffer de escrita configurado.

### 14.3 Escolhendo entre eles para "push"

| Cenário                                      | Escolha                                                     |
|----------------------------------------------|-------------------------------------------------------------|
| 100k eventos/s backend→backend               | Webhook (ou melhor: Kafka/NATS, webhook pra integração externa) |
| 100k clientes recebendo updates quase-live   | WebSocket com Redis pub/sub                                 |
| 10 eventos/dia, precisa ser confiável        | Webhook com retries                                         |
| Cliente que **responde** ao servidor em tempo real | WebSocket (só ele é bidirecional)                       |

---

## 15. Segurança

### Webhook

1. **HMAC SHA-256 no body** com segredo compartilhado. Sempre em tempo constante (`hmac.compare_digest`).
2. **Timestamp no payload** + janela (ex.: rejeitar > 5min) para evitar replay.
3. **IP allow-list** quando o produtor tem IPs estáveis (Stripe, GitHub publicam faixas).
4. **Rate limit** no endpoint.
5. **TLS obrigatório** (e `HSTS` na origem).

### WebSocket

1. **TLS (`wss://`) sempre** em produção — senão proxies podem injetar.
2. **Authenticação no handshake** via `Authorization: Bearer`, cookie de sessão ou token na query-string (menos seguro, fica em log).
3. **`Origin` check** server-side — analogia ao CORS do fetch.
4. **Rate-limit por IP/user** no accept e nas mensagens.
5. **Input validation** em cada frame recebido — parser deve ser robusto a JSON malformado.
6. **Não expor PII em broadcast** sem filtro por usuário.

---

## 16. Leituras recomendadas

- **Webhook**
  - Stripe docs — "Receiving webhooks" (um dos melhores exemplos de guia real).
  - "webhooks.fyi" — catálogo de práticas entre SaaS.
  - RFC 8935 — "Server-to-Server Event Notifications" (ainda raro, mas relevante).
- **WebSocket**
  - RFC 6455 — "The WebSocket Protocol".
  - MDN — "Writing WebSocket servers".
  - High-scale case studies: Slack, Discord e Figma publicam blog posts bons sobre escala de WS.

---

## Apêndice — cheatsheet de testes rápidos

```bash
# 1. Sobe tudo
make build && make up

# 2. Dispara um webhook para cada linguagem
make curl-webhook

# 3. Vê contadores
make stats

# 4. Abre um client WS
wscat -c ws://localhost:8001/ws

# 5. Carga de 50 req/s por 10s em todos webhooks
make burst

# 6. Um broadcast de teste em cada WS
make ws-notify

# 7. Derruba
make down
```

Boa exploração. 🐙
