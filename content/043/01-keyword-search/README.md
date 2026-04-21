# 01 — Keyword Search (BM25)

> **Teoria detalhada**: [`../THEORY.md § 2`](../THEORY.md#2-busca-léxica--bm25-o-algoritmo-do-01) — fórmula completa, PRF, inverted index, quando usar vs não usar.

Busca léxica clássica, sem nenhum modelo neural. O ranking é feito por **BM25Okapi** (Robertson/Sparck Jones, a evolução do TF-IDF usado no Lucene, Elasticsearch, OpenSearch).

## O que foi feito

- **Backend**: FastAPI em Python.
  - Carrega `docs.json` (35 documentos em PT-BR sobre tecnologias).
  - Tokeniza: lowercase + remoção de acentos via `unidecode` + split alfanumérico + remoção de stopwords PT.
  - Indexa com `rank_bm25.BM25Okapi` em memória (k1=1.5, b=0.75 default).
  - `/search?q=...&k=5` retorna hits ordenados por score BM25.
- **Frontend**: HTML/JS estático servido pela própria FastAPI (`StaticFiles`), com sugestões clicáveis.
- **Tests**: `pytest` contra a API rodando em container (assert do top-1 para queries conhecidas, score descendente, stopword-only retorna vazio).

## Como BM25 funciona (resumo rápido)

$$\text{score}(D, Q) = \sum_{q \in Q} \text{IDF}(q) \cdot \frac{f(q, D) \cdot (k_1 + 1)}{f(q, D) + k_1 \cdot (1 - b + b \cdot |D|/\text{avgdl})}$$

- Premia termos **raros no corpus** (IDF) e **frequentes no documento** (TF).
- Normaliza por tamanho do documento (`b`) para não favorecer textos longos.
- **Não entende semântica**: "banco" como instituição financeira vs. estrutura de dados é a mesma palavra.

## Como reproduzir

```bash
cd 01-keyword-search
docker compose up -d --build
# abra http://localhost:18001 no navegador
```

### Rodar os testes

```bash
./run_tests.sh
```

Equivalente a: subir o compose, aguardar healthcheck, `docker compose exec backend pytest tests/ -v`.

### Parar

```bash
docker compose down
```

## Exemplos de query

| Query | Hit esperado | Por quê |
|---|---|---|
| `banco de grafos para detecção de fraude` | Neo4j | match exato de "grafos", "fraude" |
| `orquestração de containers` | Kubernetes | "orquestração" e "containers" raros no corpus |
| `linguagem compilada com goroutines` | Go | "goroutines" aparece só em um doc |
| `cache em memória` | Redis | "cache" + "memória" |

## Limitações que motivam os próximos exemplos

- `busca vetorial` **não** encontraria "Qdrant" se o usuário perguntasse por "armazenar embeddings para similaridade" — palavras diferentes, mesmo conceito.
- Busca exata de sinônimos não funciona: "container" × "contêineres" × "imagem Docker" são tratados como tokens disjuntos.
- Isso é exatamente o que a **busca vetorial** (02) resolve.

## Estrutura

```
01-keyword-search/
├── docker-compose.yml
├── run_tests.sh
├── README.md
└── backend/
    ├── Dockerfile
    ├── requirements.txt
    ├── app.py
    ├── docs.json
    ├── frontend/
    │   └── index.html
    └── tests/
        └── test_api.py
```
