# 02 — Vector Search (busca semântica)

> **Teoria detalhada**: [`../THEORY.md § 3`](../THEORY.md#3-busca-vetorial--embeddings-densos-o-algoritmo-do-02) — contrastive learning, bi-encoder, HNSW, curse of dimensionality, quando usar.

Troca tokens por **embeddings densos**. Cada documento vira um vetor em R^384 via `intfloat/multilingual-e5-small`, e a query também. A similaridade é **cosseno**, calculada pelo **Qdrant** com índice HNSW.

## O que foi feito

- **Qdrant** (container `qdrant/qdrant:v1.12.4`) como banco vetorial, persistência em volume.
- **Backend** FastAPI com GPU (CUDA 12.4 via imagem PyTorch):
  - Carrega `intfloat/multilingual-e5-small` em CUDA.
  - No boot: espera Qdrant ficar up, cria a collection (dim 384, cosine), gera embeddings dos 35 docs e faz upsert.
  - `/search?q=...`: prefixa `"query: "`, embeda, consulta top-k no Qdrant.
  - Convenção do modelo e5: `passage: ` para docs, `query: ` para perguntas.
- **Frontend** HTML/JS com sugestões que exibem queries **onde BM25 falha** (paráfrases, sinônimos).
- **Tests**: pytest verifica que paráfrases retornam o doc alvo (Qdrant para "guardar vetores…", Terraform para "provisionar nuvem declarativa…", etc).

## Qdrant × FAISS × Chroma — por que Qdrant aqui

| | Qdrant | FAISS | Chroma |
|---|---|---|---|
| Serviço out-of-process | ✅ Docker | ❌ biblioteca | ✅ Docker |
| Filtros por payload | ✅ | parcial | ✅ |
| Escrito em | Rust | C++ | Python |
| Bom para aprender REST | ✅ | não expõe HTTP | ✅ |

## Como reproduzir

```bash
cd 02-vector-search
docker compose up -d --build      # 1ª vez: baixa imagem PyTorch (~2GB) + modelo e5
# abra http://localhost:18002
```

O primeiro boot demora ~2-3 min: download da imagem PyTorch CUDA, download do modelo e5, geração dos embeddings iniciais. A cache fica em volume (`hf-cache`), então reinícios subsequentes são rápidos.

### Rodar os testes

```bash
./run_tests.sh
```

### Verificar GPU sendo usada

```bash
docker compose logs backend | grep device
# esperado: [boot] device=cuda model=intfloat/multilingual-e5-small
```

### Inspecionar Qdrant

- Dashboard: http://localhost:16333/dashboard
- API: `curl http://localhost:16333/collections/docs`

## Exemplos onde vector >> keyword

| Query | BM25 (01) | Vector (02) |
|---|---|---|
| `onde guardo vetores para similaridade` | mistura de docs com palavra "vetor" (poucos) | **Qdrant** no topo |
| `ferramenta para provisionar nuvem declarativamente` | não tem "terraform", "provisionar", "nuvem" nos docs | **Terraform** no topo |
| `sistema de filas com garantia de entrega` | "filas" não aparece no doc de RabbitMQ | **RabbitMQ** no topo |

## Limitações que motivam o 03

- Vector search perde **matches exatos**: query `SQLite` pode não rankear SQLite no top-1 se muitos docs compartilham contexto semântico de "banco de dados".
- Tokens raros (nomes próprios, siglas, versões) são melhor tratados por BM25.
- Solução: **combinar os dois** via Reciprocal Rank Fusion → pasta `03-hybrid-search`.

## Estrutura

```
02-vector-search/
├── docker-compose.yml
├── run_tests.sh
├── README.md
└── backend/
    ├── Dockerfile           # base pytorch/pytorch:2.5.1-cuda12.4
    ├── requirements.txt
    ├── app.py               # ingest + /search
    ├── docs.json
    ├── frontend/index.html
    └── tests/test_api.py
```
