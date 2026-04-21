# 05 — RAG (Retrieval-Augmented Generation)

> **Teoria detalhada**: [`../THEORY.md § 6`](../THEORY.md#6-rag--retrieval-augmented-generation-o-algoritmo-do-05) — lost-in-the-middle, anti-hallucination, chunking, RAG × fine-tuning, avaliação (Ragas).

Finalmente, a peça que muita gente chama de "RAG" mas que na verdade é o **último passo**: depois de todo o trabalho dos exemplos 01-04 para recuperar os documentos certos, um LLM lê esse contexto e escreve uma resposta em linguagem natural com **citações**.

## Arquitetura

```
query
  ├─→ BM25 (sparse)            ┐
  └─→ e5 embedding → Qdrant    ┘─→ RRF (pool 20)
                                   │
                                   ▼
                               cross-encoder rerank (bge-reranker-v2-m3)
                                   │
                                   ▼ top-4
                               prompt builder (numera os trechos)
                                   │
                                   ▼
                               Ollama + qwen2.5:3b (GPU)
                                   │
                                   ▼
                               answer + sources (JSON)
```

## O que foi feito

- **3 serviços** no compose:
  - `qdrant` — banco vetorial.
  - `ollama` — servidor do LLM, usa GPU.
  - `backend` — FastAPI com todos os passos (embed, BM25, RRF, rerank, orquestração com Ollama).
- **Auto-pull do LLM**: na inicialização, o backend chama `/api/pull` no Ollama se `qwen2.5:3b` ainda não está baixado. Você não precisa rodar nada manualmente.
- **System prompt** força o LLM a:
  - responder só com base nos trechos numerados,
  - citar as fontes em colchetes `[1]`, `[2]`, …,
  - dizer "não sei" em vez de inventar quando a resposta não está no contexto.
- **Endpoint `/ask`** retorna `answer`, `sources` (com `rerank_score`) e três timings (`ms_retrieve`, `ms_rerank`, `ms_llm`).
- **Frontend** mostra a resposta em destaque, os trechos-fonte com índice `[N]` e o breakdown de latência.
- **Tests** validam:
  - resposta não vazia + fontes,
  - fonte correta para questão específica (Neo4j para fraude em grafos),
  - resposta contém citações com colchetes,
  - **recusa** a responder pergunta fora do domínio ("receita de brigadeiro") em vez de alucinar.

## Modelo LLM: qwen2.5:3b

- 3 bilhões de parâmetros, ~2GB em 4-bit.
- Multilingual, bom em PT-BR.
- Inference a ~50-80 tokens/s na RTX 4070 Ti SUPER.
- Trocar por outro: basta editar `LLM_MODEL` no compose e subir. Sugestões:
  - `llama3.2:3b` — similar em tamanho/velocidade.
  - `qwen2.5:7b` — respostas mais sólidas, mas ~2x mais lento.
  - `llama3.2:1b` — ultra-rápido, qualidade ruim para PT.

## Como reproduzir

```bash
cd 05-rag
docker compose up -d --build
# 1ª vez: ~5-10 min (baixa imagem PyTorch + embedding + reranker + qwen2.5:3b)
# abra http://localhost:18005
./run_tests.sh   # roda os 6 testes E2E
```

Boot logs úteis:

```bash
docker compose logs -f backend   # acompanha o boot
# esperado: [boot] pulling qwen2.5:3b → [boot] pull done → [boot] ready
```

### Exemplos de pergunta

| Query | O que acontece |
|---|---|
| "qual banco para detectar fraude em grafos?" | Retrieve acha Neo4j top-1, LLM explica por quê, cita `[1]`. |
| "como monitorar métricas?" | Traz Prometheus + Grafana, resposta menciona o par. |
| "qual a receita de brigadeiro?" | Todos os docs são sobre tech — LLM deve recusar ("não sei / não encontro"). |

### Parar e limpar

```bash
docker compose down -v   # também remove os modelos Ollama (~2GB)
```

## Quando RAG é a escolha certa?

**Sim** quando:
- A base de conhecimento muda com frequência (docs internos, tickets, wikis).
- Você precisa de citações verificáveis, não de "o modelo disse que…".
- O domínio é específico e o LLM genérico desconhece.

**Não** quando:
- A pergunta é criativa ou de opinião — RAG amarra o modelo em fatos.
- O corpus cabe no context window e a latência é crítica — jogue tudo no prompt direto.

## Limitações do que está aqui

- **Sem chunking**: cada doc é curto (~50 palavras), então a unidade de recuperação já é o doc inteiro. Para textos longos (manuais, artigos), você precisa quebrar em chunks de 200-500 tokens com overlap.
- **Sem histórico de conversa**: cada `/ask` é stateless. Para chat multi-turno: reescrever a query do usuário com base no histórico antes de recuperar.
- **Sem guardrails de saída**: o system prompt confia no LLM para citar. Um modelo maior ou um validator pós-LLM ajudaria.

## Estrutura

```
05-rag/
├── docker-compose.yml     # qdrant + ollama + backend, todos com GPU
├── run_tests.sh
├── README.md
└── backend/
    ├── Dockerfile
    ├── requirements.txt
    ├── app.py             # pipeline completo + /ask
    ├── docs.json
    ├── frontend/index.html
    └── tests/test_api.py  # E2E contra o LLM
```
