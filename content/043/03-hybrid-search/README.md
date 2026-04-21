# 03 — Hybrid Search (BM25 + vector via RRF)

> **Teoria detalhada**: [`../THEORY.md § 4`](../THEORY.md#4-busca-híbrida--rrf-o-algoritmo-do-03) — derivação do RRF, por que `k=60`, alternativas (CombSUM, Borda, learned fusion).

Combina os dois sinais do 01 e 02 usando **Reciprocal Rank Fusion**, a forma mais simples e robusta de fundir listas ranqueadas sem ter que calibrar scores de naturezas diferentes (BM25 é não-limitado, cosine é [-1, 1]).

## Fórmula (RRF — Cormack, Clarke, Buettcher 2009)

Para cada doc `d` que apareceu em **alguma** das listas:

$$\text{RRF}(d) = \sum_{\ell \in \{bm25, vec\}} \frac{1}{k + \text{rank}_\ell(d)}$$

Com `k = 60` (valor canônico do paper). Documentos ausentes de uma lista contribuem 0 naquela. O score **não depende** das magnitudes — só das posições.

## O que foi feito

- Backend faz **as duas buscas em paralelo-lógico** sobre o mesmo corpus:
  - BM25 em memória (tokenizer com unidecode + stopwords PT).
  - Dense via Qdrant + `multilingual-e5-small`.
- Fetch um **pool** maior (default 20) de cada, funde com RRF e corta no top-k.
- `/search?q=...&mode=hybrid|bm25|vector&k=5&pool=20`: o parâmetro `mode` permite comparar lado a lado.
- Cada hit híbrido retorna `bm25_rank` e `vector_rank` → dá pra ver **por qual caminho** o doc veio.
- Frontend com toggle entre os 3 modos.

## Quando o hybrid ganha?

| Query | BM25 (01) | Vector (02) | Hybrid (03) |
|---|---|---|---|
| `sistema de filas com ack e garantia de entrega` | RabbitMQ #1 (literal "AMQP"/"ack") | RabbitMQ #4 (Redis/Helm vieram antes) | **RabbitMQ no topo** |
| `onde guardo vetores para similaridade` | Qdrant não ranqueia (sem tokens em comum) | Qdrant #1 | Qdrant #1 |
| `linguagem compilada goroutines` | Go #1 | Go #1 | Go #1 (ambos concordam) |
| `ferramenta declarativa para provisionar nuvem` | 0 matches relevantes | Terraform #1 | Terraform #1 |

RRF é **anti-frágil**: só precisa que **um** dos dois dê match decente.

## Como reproduzir

```bash
cd 03-hybrid-search
docker compose up -d --build
# abra http://localhost:18003
./run_tests.sh   # roda a suite que valida cada modo
```

### Parar

```bash
docker compose down -v    # -v remove os volumes (qdrant-data, hf-cache)
```

## Por que RRF em vez de combinação linear?

Você poderia tentar `score = α · bm25 + (1-α) · cos`. Problema:

- BM25 varia de 0 a 15+ dependendo da query, cosine só de -1 a 1.
- Precisa normalizar (min-max? softmax?) e tunar α por corpus.
- Se adicionar um 3º signal (reranker, por exemplo), tudo recalibra.

RRF não liga para magnitudes — é **ordinal**. Adicionar um terceiro canal é só somar mais um termo. Por isso é o default de produção de Elastic, Azure AI Search, Vespa, Weaviate.

## Limitação que motiva o 04

RRF **não conhece a semântica da query**: ele apenas acredita nos dois rankings iniciais. Se os **dois** estavam errados (um falso-positivo no topo por coincidência léxica + semântica), o doc errado continua no topo.

Solução: após o hybrid retornar, rodar um **cross-encoder** que olha query e doc juntos (em vez de em bi-encoder) e reordena o pool. Isso é o **reranking** → pasta `04-reranking`.

## Estrutura

```
03-hybrid-search/
├── docker-compose.yml
├── run_tests.sh
├── README.md
└── backend/
    ├── Dockerfile
    ├── requirements.txt
    ├── app.py              # BM25 + vector + RRF + /search?mode=...
    ├── docs.json
    ├── frontend/index.html # toggle hybrid|bm25|vector
    └── tests/test_api.py
```
