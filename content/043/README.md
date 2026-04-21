# 043 — Busca: Keyword → Vetorial → Híbrida → Reranking → RAG

> **📘 Teoria completa → [`THEORY.md`](THEORY.md)** — glossário, fórmulas, algoritmos e matriz de decisão "quando usar cada um".

Cinco exemplos **independentes**, cada um em uma subpasta com seu próprio `docker-compose.yml`, backend FastAPI, frontend HTML+JS e suite de testes. O mesmo corpus de 35 documentos em PT-BR é usado em todos — assim dá para rodar a **mesma query** nos cinco e ver como cada técnica muda o resultado.

## Progressão

| # | Pasta | Adiciona | Stack | GPU |
|---|---|---|---|---|
| 01 | [keyword-search](01-keyword-search/README.md) | BM25 sobre tokens normalizados | FastAPI + rank_bm25 | ❌ |
| 02 | [vector-search](02-vector-search/README.md) | Embeddings densos, similaridade de cosseno | FastAPI + Qdrant + multilingual-e5 | ✅ |
| 03 | [hybrid-search](03-hybrid-search/README.md) | Fusão BM25 + vector via Reciprocal Rank Fusion | 01 + 02 juntos | ✅ |
| 04 | [reranking](04-reranking/README.md) | Cross-encoder reordena top-20 do hybrid | 03 + bge-reranker-v2-m3 | ✅ |
| 05 | [rag](05-rag/README.md) | LLM lê top-k e gera resposta com citações | 04 + Ollama (qwen2.5:3b) | ✅ |

Cada pasta é **autossuficiente** — não importa nada da pasta anterior. A ideia é ler/rodar em ordem para ver a ideia crescendo, mas cada compose sobe sozinho.

## Pré-requisitos

- Docker + Docker Compose v2.
- GPU NVIDIA com `nvidia-container-toolkit` (para 02-05).
- Testado em RTX 4070 Ti SUPER 16GB, Python 3.12 no host (só para leitura — dentro dos containers é 3.11).
- Sem GPU: 01 continua funcionando. 02-05 caem pra CPU se você remover o bloco `deploy.resources.reservations.devices` (mas ficam lentos).

## Portas (para rodar todos simultaneamente sem conflito)

| Serviço | Porta host |
|---|---|
| 01 backend | 18001 |
| 02 backend / qdrant | 18002 / 16333 |
| 03 backend / qdrant | 18003 / 16334 |
| 04 backend / qdrant | 18004 / 16335 |
| 05 backend / qdrant / ollama | 18005 / 16336 / 11435 |

## Dataset

`shared/docs.json` — 35 documentos curtos em PT-BR sobre bancos de dados, linguagens, infraestrutura e observabilidade. Cada pasta recebe uma cópia no build para manter independência.

## Como rodar tudo (em ordem)

```bash
# opção 1: um de cada vez, lendo o README de cada um
cd 01-keyword-search && ./run_tests.sh && docker compose down && cd ..
cd 02-vector-search  && ./run_tests.sh && docker compose down && cd ..
cd 03-hybrid-search  && ./run_tests.sh && docker compose down && cd ..
cd 04-reranking      && ./run_tests.sh && docker compose down && cd ..
cd 05-rag            && ./run_tests.sh && docker compose down && cd ..
```

Primeira vez leva ~15-20 min no total (download de imagem PyTorch + modelos). Cache fica em volumes Docker (`hf-cache`, `ollama-models`, `qdrant-data`) — rodar de novo é questão de segundos.

## Modelos usados

| Papel | Modelo | Tamanho | Onde |
|---|---|---|---|
| Embeddings | `intfloat/multilingual-e5-small` | ~470MB | 02, 03, 04, 05 |
| Reranker (cross-encoder) | `BAAI/bge-reranker-v2-m3` | ~2.2GB | 04, 05 |
| LLM gerador | `qwen2.5:3b` (4-bit via Ollama) | ~2GB | 05 |

Todos **multilingual** — escolhidos especificamente para lidar bem com PT-BR.

## Ideia central em uma frase por exemplo

- **01** — "Se o termo literal aparece no doc, eu acho."
- **02** — "Se o doc fala sobre o mesmo **assunto**, eu acho (mesmo sem nenhuma palavra em comum)."
- **03** — "Some os ranks. Um acerto numa das duas basta."
- **04** — "Olha query e doc **juntos** com cross-attention para ordenar direito o top-20."
- **05** — "Agora que o top-k está bom, deixa o LLM escrever a resposta e citar de onde tirou."

## Troubleshooting rápido

**`port is already allocated`**  
Alguma porta (18001-18005, 16333-16336, 11435) está ocupada por outro projeto. Ajuste no `docker-compose.yml` da pasta ou derrube o outro container.

**Backend fica `unhealthy` no 02-05**  
Quase sempre é a primeira execução ainda baixando os modelos. Acompanhe:  
```bash
docker compose logs -f backend
```
Procure por linhas `[boot] ingested` (ingestão pronta) e `[boot] ready` (pipeline pronto).

**`nvidia` driver not found**  
Falta o `nvidia-container-toolkit`. No Ubuntu:  
```bash
sudo apt install nvidia-container-toolkit
sudo systemctl restart docker
docker run --rm --gpus all nvidia/cuda:12.3.1-base-ubuntu22.04 nvidia-smi
```

**Testes do 05 dando timeout**  
Aumente o timeout no `test_api.py` — inference local do LLM pode pegar 10-20s em queries longas.

## Estrutura geral

```
043/
├── README.md                          # este arquivo
├── shared/
│   └── docs.json                      # corpus fonte (copiado em cada backend/)
├── 01-keyword-search/
├── 02-vector-search/
├── 03-hybrid-search/
├── 04-reranking/
└── 05-rag/
```
