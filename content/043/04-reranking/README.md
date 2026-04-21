# 04 — Reranking (cross-encoder)

> **Teoria detalhada**: [`../THEORY.md § 5`](../THEORY.md#5-reranking--cross-encoder-o-algoritmo-do-04) — arquitetura cross-attention, treino em MS-MARCO, distilação, listwise vs pointwise.

Em cima do pipeline híbrido (03), adiciona uma etapa final: um **cross-encoder** (`BAAI/bge-reranker-v2-m3`) reavalia os top-20 do RRF observando **query e documento juntos**, e devolve um score de relevância mais preciso. Isso consegue arranjar no topo um doc que o bi-encoder empurrou pro #7.

## Bi-encoder × cross-encoder (o ponto central)

|  | Bi-encoder (02, 03) | Cross-encoder (04) |
|---|---|---|
| Input | `query` e `doc` **separados** | `(query, doc)` no **mesmo input** |
| Embedding | 1 vetor por doc, pré-calculado | não tem embedding — cada par é avaliado a cada request |
| Latência | O(log N) via HNSW | O(k), mas com k pequeno (20) |
| Relevância | boa (similaridade no espaço latente) | **melhor** (cross-attention entre tokens de query e doc) |

Regra prática: bi-encoder para recall (pegar os top-N candidatos), cross-encoder para precision (ordenar esses N corretamente).

## O que foi feito

- Reaproveita BM25 + Qdrant + RRF do 03.
- Adiciona `CrossEncoder("BAAI/bge-reranker-v2-m3")` do sentence-transformers, em CUDA.
- `/search?mode=rerank`: busca o pool híbrido (20 docs), alimenta `(query, "title. content")` no reranker em batch, reordena por score.
- `/search?mode=hybrid`: pula o rerank, para comparar lado a lado na UI.
- Frontend mostra duas colunas (hybrid × reranked), com indicador `↑N` / `↓N` mostrando quanto cada doc subiu ou desceu depois do rerank.
- Tests validam: monotonicidade do score, cobertura do top-1 original, query que **especificamente** melhora com rerank (React para "biblioteca que renderiza componentes").

## Reranker escolhido: bge-reranker-v2-m3

- Multilingual (funciona bem em PT-BR, 100+ idiomas).
- ~568M params, ~2.2GB, cabe folgado em GPUs de ≥8GB.
- Saída não é probabilidade — é um logit; usar só para **ordenar**, não como limiar absoluto.
- Alternativas: `BAAI/bge-reranker-base` (mais rápido, só en/zh), `cross-encoder/ms-marco-MiniLM-L-6-v2` (inglês, pequeno).

## Como reproduzir

```bash
cd 04-reranking
docker compose up -d --build
# 1ª vez: ~300s (baixa imagem PyTorch + embedding model + reranker ~2.2GB)
# abra http://localhost:18004
./run_tests.sh
```

### Comparar hybrid × rerank na UI

- Digite `biblioteca que renderiza componentes no navegador` → repare que React sobe no lado direito.
- Digite `monitoramento pull-based de métricas HTTP` → Prometheus deve estar top-1 em ambos.

### Latência

Na RTX 4070 Ti SUPER, rerank de 20 docs fica em ~30-80ms (incluindo round-trip HTTP). CPU inference seria ~2-4s — por isso **reranker é onde GPU paga mais**.

### Parar

```bash
docker compose down -v
```

## Quando vale a pena adicionar reranker?

- Você já tem recall bom (hybrid está retornando os docs certos no top-20) mas a ordem está ruim.
- Sua aplicação é sensível a precisão (RAG, FAQ, assistente jurídico).
- Latência extra (~50ms na GPU) é aceitável.

**Não vale a pena** se o recall já está ruim — reranker não inventa doc, só reordena. Nesse caso foque em melhorar embeddings/chunking antes.

## Próximo: 05-rag

Com a precisão do top-k garantida, finalmente dá pra alimentar esse contexto em um LLM para gerar uma resposta em linguagem natural com citações → pasta `05-rag`.

## Estrutura

```
04-reranking/
├── docker-compose.yml
├── run_tests.sh
├── README.md
└── backend/
    ├── Dockerfile
    ├── requirements.txt
    ├── app.py               # hybrid + cross-encoder
    ├── docs.json
    ├── frontend/index.html  # lado a lado hybrid × reranked
    └── tests/test_api.py
```
