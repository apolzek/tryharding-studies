# 042 — Design Patterns em Go

Implementações de 26 padrões de projeto em Go, cada um em cenário próximo de produção, com testes (incluindo `-race` para os concorrentes) e documentação detalhada em português.

Cada pasta é um módulo Go independente e auto-contido — não há dependência entre elas.

## Como rodar qualquer pattern

```bash
cd 042/<pasta-do-pattern>
go run .                    # executa a demonstração
go test -race -v ./...      # executa os testes
go test -race -cover ./...  # testes com cobertura
```

## Índice

### Criacionais — como objetos são instanciados

| # | Pattern | Cenário |
|---|---------|---------|
| 01 | [Singleton](01-singleton/README.md) | Pool global de conexões com banco via `sync.Once` |
| 02 | [Factory Method](02-factory-method/README.md) | Fábrica de gateways de pagamento (Stripe, PayPal, Pix) |
| 03 | [Abstract Factory](03-abstract-factory/README.md) | Provedor cloud multi-região (AWS/GCP) produzindo Storage + Queue |
| 04 | [Builder](04-builder/README.md) | Builder fluente de requisições HTTP |
| 05 | [Prototype](05-prototype/README.md) | Clonagem de contratos/templates com deep copy |

### Estruturais — como objetos se compõem

| # | Pattern | Cenário |
|---|---------|---------|
| 06 | [Adapter](06-adapter/README.md) | Adaptar gateway legado SOAP/XML à interface REST moderna |
| 07 | [Bridge](07-bridge/README.md) | Notificações desacoplando Alert × Canal (Email/SMS/Slack) |
| 08 | [Composite](08-composite/README.md) | Organograma corporativo com Employee + Department |
| 09 | [Decorator](09-decorator/README.md) | Middlewares HTTP (logging, auth, rate-limit, metrics) |
| 10 | [Facade](10-facade/README.md) | Checkout orquestrando estoque + pagamento + envio + notificação |
| 11 | [Proxy](11-proxy/README.md) | Cache em memória + rate-limit em frente a API externa |

### Comportamentais — como objetos interagem

| # | Pattern | Cenário |
|---|---------|---------|
| 12 | [Chain of Responsibility](12-chain-of-responsibility/README.md) | Pipeline HTTP: auth → rate-limit → schema → regras de negócio |
| 13 | [Command](13-command/README.md) | Fila de tarefas com undo em operações reversíveis |
| 14 | [Iterator](14-iterator/README.md) | Iterator cursor-based sobre API paginada |
| 15 | [Mediator](15-mediator/README.md) | Sala de leilão com lances e regras centralizadas |
| 16 | [Observer](16-observer/README.md) | Event bus thread-safe com subscribers assíncronos |
| 17 | [State](17-state/README.md) | Máquina de estados do ciclo de vida de Pedido |
| 18 | [Strategy](18-strategy/README.md) | Cálculo de frete com estratégias intercambiáveis |
| 19 | [Template Method](19-template-method/README.md) | Pipeline de relatórios (CSV/JSON/PDF) compartilhando esqueleto |
| 20 | [Visitor](20-visitor/README.md) | AST de expressões com Evaluator, Printer e Optimizer |
| 21 | [Memento](21-memento/README.md) | Editor com undo/redo via snapshots opacos |

### Concorrência — idiomáticos do Go

| # | Pattern | Cenário |
|---|---------|---------|
| 22 | [Worker Pool](22-worker-pool/README.md) | Pool de N workers com shutdown limpo via `context` |
| 23 | [Pipeline](23-pipeline/README.md) | Ingestão → parse → enrich → persist em goroutines encadeadas |
| 24 | [Fan-out / Fan-in](24-fan-out-fan-in/README.md) | Agregador paralelo de cotações de múltiplos brokers |
| 25 | [Semaphore](25-semaphore/README.md) | Limitador de concorrência em chamadas a API externa |
| 26 | [Generator](26-generator/README.md) | Streams preguiçosos (paginação + tokens infinitos) canceláveis |

## Estrutura de cada pasta

```
<NN>-<pattern>/
├── README.md            # documentação detalhada em português
├── go.mod               # módulo independente (module patterns/<slug>)
├── main.go              # demonstração executável
├── <slug>.go            # implementação do pattern
└── <slug>_test.go       # testes table-driven
```

## Verificar tudo de uma vez

```bash
# roda build + testes em todas as 26 pastas
cd 042
for d in */; do
  (cd "$d" && go build ./... && go test -race -count=1 ./...)
done
```

## Filosofia das implementações

- **Exemplo realista**: cada pattern usa um domínio que você provavelmente já viu em produção (pagamento, checkout, notificação, logs, cotação).
- **Stdlib apenas**: nenhuma dependência externa — todo o código depende só da biblioteca padrão.
- **Idiomático**: composição sobre herança, interfaces pequenas, erros explícitos, `context.Context` em I/O.
- **Testes com `-race`**: todos os patterns concorrentes (e os não-concorrentes também) rodam com o race detector.
- **README com trade-offs**: cada pattern inclui seção explícita de "Quando NÃO usar" — armadilhas comuns e sinais de over-engineering.
