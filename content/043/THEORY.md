# Teoria — termos, algoritmos e quando usar cada busca

Um guia denso mas lido de ponta a ponta. Cada conceito está amarrado em um dos 5 demos (`01`…`05`), então você pode sempre voltar ao código e **ver** a ideia funcionando.

---

## 1. Glossário — termos que aparecem o tempo todo

### Retrieval × Ranking × Generation

- **Retrieval**: dado uma query, devolver um **conjunto** (pool) de candidatos. Foco em **recall** — não perder docs relevantes.
- **Ranking**: dado um conjunto pequeno já filtrado, **ordenar** por relevância. Foco em **precision@k** — colocar o certo no topo.
- **Generation**: dado contexto + pergunta, escrever uma resposta em linguagem natural. É o que um LLM faz no RAG (05).

Um pipeline moderno é **retrieval → ranking → generation**. Os 5 demos montam essa cadeia peça por peça.

### Recall × Precision × F1 (a métrica mais usada em busca)

- **Recall@k** — dos docs relevantes que existem, quantos apareceram nos primeiros `k` resultados? Importa em retrieval.
- **Precision@k** — dos `k` resultados retornados, quantos são relevantes? Importa em ranking.
- **MRR** (Mean Reciprocal Rank) — média de `1/posição_do_primeiro_relevante`. Importa quando só o top-1 interessa (ex.: autocomplete).
- **nDCG** (normalized Discounted Cumulative Gain) — pondera a posição (acertar no #1 vale mais que no #5) e aceita graus de relevância (não só 0/1).

Regra prática: **retrieval optimiza recall, ranking optimiza nDCG/precision**.

### Sparse × Dense

- **Sparse** (BM25, TF-IDF): cada doc é um vetor com **milhares** de dimensões (uma por termo do vocabulário), mas **99.99% são zero**. Só entram as palavras que o doc **realmente** contém. Busca é por interseção de termos → **inverted index** (lista de docs que contêm a palavra `X`).
- **Dense** (embeddings): cada doc vira um vetor **denso** de ~384-1024 dimensões, todas com valores. Busca é por similaridade no espaço latente.

Complementares — é isso que o 03 (hybrid) explora.

### Bi-encoder × Cross-encoder

A distinção mais importante pra entender 02→04:

- **Bi-encoder** (02, 03): query e doc passam **separadamente** pelo modelo, cada um vira um vetor. Comparação final é **similaridade de cosseno**. Cada doc é codificado **uma vez** e indexado.
- **Cross-encoder** (04): query e doc são **concatenados** num único input `[CLS] query [SEP] doc [SEP]`, e o modelo retorna **um único score** observando as duas partes juntas via self-attention. **Cada par** precisa ser calculado no momento da query — não dá para pré-indexar.

```
Bi-encoder:      encode(query) · encode(doc_i)           # N operações independentes
Cross-encoder:   encode(query, doc_i)                     # precisa re-rodar para cada doc
```

Regra prática: bi-encoder para **recall** (barato em N), cross-encoder para **precision** (caro, mas só no top-K).

### Embedding, dimensão e distância

- **Embedding** = vetor numérico que representa um texto no espaço latente. Texts próximos em significado → vetores próximos.
- **Dimensão** = tamanho do vetor. Comum: 384 (small), 768 (base), 1024 (large), 1536 (OpenAI `text-embedding-3-small`).
- **Distância** entre vetores:
  - **Cosseno**: `cos(u, v) = (u · v) / (||u|| · ||v||)` → foca na **direção**, ignora magnitude. É o default de busca semântica.
  - **Dot product**: `u · v` → funciona se os vetores estão normalizados (equivalente ao cosseno nesse caso).
  - **L2 (euclidiana)**: `||u − v||` → raramente usada em texto.
- **Normalização**: dividir cada vetor por `||v||` para virar unitário. Faz cosseno = dot product. O e5 já retorna normalizado com `normalize_embeddings=True` (demos 02-05).

### Logits, softmax, temperatura (aparece no reranker e no LLM)

- **Logit** = output cru do modelo antes de virar probabilidade. No bge-reranker é o logit do token `<score>` — compara com outros logits, **não interprete como probabilidade**.
- **Softmax**: converte vetor de logits em distribuição de probabilidade.
- **Temperatura**: divide os logits antes do softmax. `T<1` → distribuição mais pontuda (determinístico). `T>1` → mais uniforme (criativo). No 05 usamos `T=0.1` pro LLM responder sempre a mesma coisa pra mesma pergunta.

### Chunking (relevante para RAG)

Quebrar documentos longos em pedaços menores antes de indexar. Tradeoffs:

- **Chunk muito pequeno** (100 tokens): perde contexto, o retrieval acha frase isolada que pode significar outra coisa.
- **Chunk muito grande** (2000 tokens): match fica diluído — a parte relevante é 10% do chunk, o resto é ruído no cosseno.
- **Overlap** (ex.: 20%): chunks se sobrepõem para não cortar informação no meio.

Nos demos os "docs" já estão no tamanho certo (~50 palavras), por isso não chunkamos. Em produção com artigos, PDFs, manuais: chunking é quase sempre necessário.

### Context window, top-k, top-p

- **Context window**: quantos tokens o LLM consegue ler numa chamada. Qwen2.5:3b tem 128k, mas o custo cresce com o tamanho.
- **Top-k retrieval**: quantos docs passar pro LLM. No 05 usamos `top_k=4`.
- **Top-p (nucleus sampling)**: no LLM, amostra tokens até acumular probabilidade `p`. Desacoplado de temperatura.

### Latency budget

Orçamento de tempo total da busca. Tipicamente:

| Cenário | Budget | O que cabe |
|---|---|---|
| Autocomplete | <50ms | só BM25 ou só vector |
| Busca interativa | <300ms | hybrid sem rerank |
| Assistente conversacional | 1-5s | hybrid + rerank + LLM |
| Análise assíncrona | sem limite | LLM por doc, reranking com LLM |

---

## 2. Busca léxica — BM25 (o algoritmo do 01)

### De onde veio

TF-IDF (1972) → **Probabilistic Relevance Framework** (Robertson, 1976) → **Okapi BM25** (Robertson & Sparck Jones, 1994). É o ranking que o Lucene, Elasticsearch, OpenSearch, Solr e Tantivy usam **por default** até hoje.

### Fórmula

$$\text{BM25}(D, Q) = \sum_{q \in Q} \text{IDF}(q) \cdot \frac{f(q, D) \cdot (k_1 + 1)}{f(q, D) + k_1 \cdot (1 - b + b \cdot |D|/\text{avgdl})}$$

Decodificando:

- `f(q, D)` — quantas vezes o termo `q` aparece no doc `D` (Term Frequency).
- `IDF(q) = log((N − df + 0.5) / (df + 0.5) + 1)` — inverse document frequency. Termo que aparece em poucos docs do corpus tem IDF alto → é informativo. "de", "o", "que" têm IDF baixo.
- `|D|` — tamanho do doc em palavras, `avgdl` — média dos tamanhos no corpus.
- `k_1` (default 1.5) — quão rápido TF satura. Com `k_1=0` só importa se o termo existe (binário). Com `k_1` grande, repetir a palavra cresce linearmente.
- `b` (default 0.75) — penalização por doc ser longo. `b=0` desliga, `b=1` normaliza totalmente por tamanho.

### Por que funciona

Saturação do TF + normalização por comprimento resolvem dois problemas reais do TF-IDF ingênuo:
1. Uma palavra repetida 10 vezes não é 10× mais relevante que 1 vez — BM25 **satura**.
2. Um doc de 10 mil palavras tem mais chance de casar com qualquer termo — BM25 **penaliza** o tamanho.

### Estrutura de dados: inverted index

```
"fraude"   → [doc_8 (posição 12), doc_12 (posição 3)]
"grafos"   → [doc_8 (posição 5), doc_35 (posição 8)]
"banco"    → [doc_1, doc_2, doc_6, doc_7, ..., doc_35]
```

Busca = interseção de posting lists, somando scores. Escala para bilhões de docs.

### Quando usar BM25

- **Sempre** como baseline. Se a query tem termos raros e específicos, BM25 é difícil de bater.
- Nomes próprios, siglas, códigos, versões (`CVE-2024-3094`), identificadores.
- Corpus curado onde terminologia é consistente (código-fonte, logs, documentação API).
- Sem GPU, sem budget para modelos.

### Quando NÃO usar sozinho

- Queries que **parafraseiam** — o usuário escreve "banco em memória" e o doc diz "armazenamento volátil". Tokens disjuntos → zero match.
- Multilingual — BM25 não entende que "container" e "contêineres" são o mesmo conceito.
- Perguntas de linguagem natural longas ("como faço para monitorar uma API HTTP?") — muitos tokens de ruído.

---

## 3. Busca vetorial — embeddings densos (o algoritmo do 02)

### Como o modelo aprende a codificar semântica

**Contrastive learning** é o truque central. Durante o treino, o modelo vê:
- Um par **positivo** (query, doc que responde a query).
- Vários pares **negativos** (query, docs irrelevantes — geralmente outros docs do mesmo batch).

A função de perda (InfoNCE / MultipleNegativesRankingLoss) força o modelo a deixar o positivo **mais próximo** da query que todos os negativos:

$$\mathcal{L} = -\log \frac{\exp(\text{sim}(q, d^+) / \tau)}{\exp(\text{sim}(q, d^+) / \tau) + \sum_{d^-} \exp(\text{sim}(q, d^-) / \tau)}$$

Depois de milhões desses pares, o modelo converge para um espaço onde **distância = relevância semântica**.

### Arquitetura bi-encoder (dual encoder)

```
query ──► [encoder] ──► q_vec ─┐
                                ├──► cosine(q_vec, d_vec)
doc   ──► [encoder] ──► d_vec ─┘
```

Os dois ramos compartilham pesos. Os d_vec são pré-calculados e indexados — na query só gera q_vec e busca o mais próximo.

### Por que o e5 pede prefixos `query:` e `passage:`

O `intfloat/multilingual-e5-small` foi treinado como **asymmetric dual encoder**: o encoder é o mesmo, mas query e passage recebem **prompts diferentes** durante o treino. Queries são tipicamente curtas e em forma de pergunta; passages são textos descritivos. Os prefixos dizem ao modelo "você está vendo X", e ele gera um vetor otimizado pra esse papel.

Esquecer o prefixo degrada acuracia em ~5-15%.

### Curse of dimensionality (por que 384 é melhor que 10.000)

Em alta dimensão, distâncias entre pontos aleatórios tendem a ficar todas parecidas. Isso mataria a busca. Os modelos modernos resolvem via treino contrastivo: forçam a **estrutura** do espaço, fazendo semelhanças se concentrarem em subespaços úteis. Dimensões comuns e a razão:

| Dim | Uso típico | Trade-off |
|---|---|---|
| 384 | MiniLM, e5-small | rápido, 4× menos memória que 1024, quality perde ~5% em BEIR |
| 768 | BERT-base, e5-base | default histórico |
| 1024 | e5-large, bge-large | +2-3% de quality sobre 768 |
| 1536 | OpenAI text-embedding-3-small | não muda muito sobre 1024, só API |

### ANN — Approximate Nearest Neighbor

Busca exata é O(N·D) — para 1M de docs e 384 dims é 384M mults por query. Lento. Soluções:

- **HNSW** (Hierarchical Navigable Small World) — grafo em camadas, cada nó conecta a poucos vizinhos. Busca desce do topo (poucas ligações, saltos longos) pra base (muitas ligações, ajuste fino). É o que o **Qdrant**, Milvus, Weaviate e Elasticsearch 8+ usam. Complexidade empírica O(log N), recall >95%.
- **IVF** (Inverted File Index) — clusteriza os vetores (k-means), na query procura só nos `nprobe` clusters mais próximos. Usado em FAISS.
- **PQ** (Product Quantization) — comprime cada vetor em poucos bytes aproximando-o pelo código mais próximo em codebooks. Economiza 10-100× de memória com perda controlada.
- **LSH** (Locality-Sensitive Hashing) — hashing tal que vetores similares caem no mesmo bucket. Cai em desuso.

O **Qdrant** nos demos usa HNSW por default (parâmetros `m=16`, `ef_construct=100`).

### Quando usar busca vetorial

- Queries **em linguagem natural** ("como isolar um serviço que responde lento?").
- **Paráfrases e sinônimos** críticos (e-commerce, suporte, FAQ).
- Corpus **multi-domínio** onde terminologia varia.
- **Cross-lingual** (query em PT, doc em EN) — modelos multilingual tipo e5 ou mBGE fazem isso.

### Quando NÃO usar sozinho

- Nomes próprios, siglas raras, códigos de produto — embeddings generalizam demais e podem confundir ("RAM" e "memória" ficam próximos, mas "RTX 4070 Ti SUPER" e "RTX 4090" também ficam próximos indevidamente).
- Corpus muito pequeno (<100 docs) — os embeddings dão empates semânticos, precisão cai.
- Queries curtas de uma palavra (o usuário digitou só `SQLite`) — literal tem que vencer.

---

## 4. Busca híbrida — RRF (o algoritmo do 03)

### Por que combinar

BM25 e vector **erram em lugares diferentes**. BM25 falha quando a query parafraseia; vector falha quando o usuário quer um termo **literal específico**. Combinando, você cobre os dois casos. Isso se chama **error complementarity**.

### Reciprocal Rank Fusion (RRF)

$$\text{RRF}(d) = \sum_{\ell \in \{\text{bm25, vec}\}} \frac{1}{k + \text{rank}_\ell(d)}$$

`k=60` vem do paper original (Cormack, Clarke, Buettcher, SIGIR 2009). O valor surgiu empírico em TREC — valores entre 40 e 100 dão resultados similares, 60 é a mediana.

### Por que `k=60` (e não 0)

Se `k=0`, rank 1 contribui `1`, rank 2 contribui `0.5`, rank 3 contribui `0.33` — o top-1 domina demais. Com `k=60`, rank 1 contribui `1/61 ≈ 0.0164`, rank 2 contribui `1/62 ≈ 0.0161`, rank 10 contribui `1/70 ≈ 0.0143`. **Aplanamento deliberado** — um doc que está bem em ambas as listas (digamos #5 em cada) vence um doc que está #1 em uma e fora da outra:

- A: em #5 + #5 → `1/65 + 1/65 = 0.0308`
- B: em #1 + não aparece → `1/61 = 0.0164`

**A vence**. Isso captura o princípio: concordância nas duas listas é mais forte que um pico isolado em uma.

### Por que não combinação linear

Tentar `score = α·bm25 + (1-α)·cos`:

- BM25 varia de 0 a 15+; cosine de -1 a 1. Somar direto quebra.
- Você teria que normalizar (min-max? z-score? softmax?) — cada normalização tem bugs (outliers distorcem min-max, softmax assume distribuição gaussiana).
- α é hiperparâmetro que depende de corpus e tipo de query.

RRF é **ordinal** — ignora magnitudes, só olha posições. Por isso é o default de produção do Elastic, Azure AI Search, Weaviate, Vespa.

### Alternativas a RRF

- **CombSUM / CombMNZ** — soma direta dos scores normalizados; precisa calibração.
- **Borda count** — cada doc recebe `N - rank` pontos por lista, soma. Similar a RRF mas sem `1/k`.
- **Learned fusion** — treina um modelo (LGBM, etc.) tomando features das duas listas. Ganha ~2-3% sobre RRF mas custa treino e manutenção.

### Quando hybrid vale a pena

- **Quase sempre** vale — o custo é 1 BM25 a mais e uma soma. BM25 em memória pra 1M docs é <5ms.
- Queries **mistas** (usuário digita "Terraform para AWS") — precisa literal E semântico.
- Corpus **heterogêneo** (códigos + texto descritivo).

### Quando não ajuda

- Corpus muito uniforme onde os dois métodos concordam quase sempre — hybrid = BM25 ou = vector.
- Queries puramente de uma palavra — BM25 sozinho já mata.
- Budget extremo (<10ms) — hybrid adiciona latência.

---

## 5. Reranking — cross-encoder (o algoritmo do 04)

### A arquitetura

O cross-encoder é um BERT treinado para classificar pares. Input:

```
[CLS] qual banco para fraude em grafos? [SEP] Neo4j é um banco de dados de grafos nativo... [SEP]
```

Ele passa por **self-attention** onde **cada token da query pode atender a cada token do doc e vice-versa**. A saída é um único logit que estima "quão relevante o doc é pra query".

### Por que cross-attention ganha sobre bi-encoder

No bi-encoder, cada texto é comprimido num vetor de 384 floats — uma perda enorme de informação. Depois compara com cosseno.

No cross-encoder, o modelo pode fazer raciocínios como:
- "a query pergunta sobre **detecção de fraude**; o doc menciona **detecção de fraude** explicitamente em uma frase coerente com **grafos** — alta relevância."
- "a query é sobre **grafos**; o doc menciona **grafos** mas no contexto de 'banco de dados de grafos' que é só um detalhe de sidebar — baixa relevância."

O bi-encoder não consegue essa análise fina porque os dois lados já viraram vetores **antes** de se encontrarem.

### Custo

- Bi-encoder: 1 chamada por doc na indexação, 1 chamada por query (≈ 5-10ms na GPU).
- Cross-encoder: 1 chamada **por par** (query, doc) na hora da query (≈ 2-5ms na GPU por par, em batch mais rápido por par).

Por isso o padrão é **bi-encoder (+ BM25) para recall no top-100, cross-encoder para rerank no top-20**. Não dá para cross-encoder num corpus de 1M docs em tempo real.

### Modelos

| Modelo | Idiomas | Tamanho | Notas |
|---|---|---|---|
| `cross-encoder/ms-marco-MiniLM-L-6-v2` | EN | 22M | Rápido, benchmark histórico |
| `BAAI/bge-reranker-base` | EN, ZH | 278M | Balanceado |
| `BAAI/bge-reranker-v2-m3` | 100+ (inclui PT) | 568M | **Usado no 04/05**, multilingual forte |
| `jina-reranker-v2-base-multilingual` | 30+ | 278M | Mais rápido |
| `Cohere rerank-v3` | API paga | — | Benchmark estado-da-arte |
| `LLM-as-reranker` (GPT-4o, Claude) | — | enorme | Custo alto, qualidade topo |

### Treino: MS-MARCO e distilação

A base padrão é o **MS-MARCO** (Microsoft, ~500k queries reais do Bing com passages anotados). Modelos são treinados em pares (query, relevant) vs (query, hard negative — um doc que **parece** relevante mas não é).

**Distilação**: treina um bi-encoder para imitar os scores de um cross-encoder maior (professor). Obtém ~80% da quality do cross com o custo do bi. É a ideia por trás de `cross-encoder/ms-marco-MiniLM-L-6-v2` ser derivado de BERT-large.

### Listwise × pairwise × pointwise

Como o reranker é treinado a ver relevância:

- **Pointwise** — um par (query, doc) vira um número isolado. O mais simples, usado em quase todos os cross-encoders modernos.
- **Pairwise** — o modelo vê dois docs simultaneamente e decide qual é mais relevante. RankNet.
- **Listwise** — vê a lista inteira e otimiza a ordem toda. LambdaRank, LambdaMART. Melhor teoricamente, mais complicado de treinar.

### Quando usar reranker

- Recall já está bom (os docs certos estão no top-20), mas a **ordem** importa (FAQ, assistente, RAG).
- Budget de latência aceita +30-80ms.
- Tem GPU — em CPU um cross-encoder é ~50× mais lento.
- Quer ganho sem re-treinar embeddings.

### Quando não vale

- Recall está ruim — rerank não inventa, só reordena. Investe em chunking/embeddings primeiro.
- Latência crítica (<100ms total).
- Top-1 é suficiente E bi-encoder já acerta ele.

---

## 6. RAG — Retrieval-Augmented Generation (o algoritmo do 05)

### Por que existe

LLMs **sozinhos** têm 3 problemas:

1. **Conhecimento congelado** no cutoff de treino.
2. **Hallucination** — inventam fatos plausíveis quando não sabem.
3. **Não citam fontes** — resposta é opaca.

RAG resolve os três: o LLM só vê a pergunta **junto** com trechos recuperados do **seu** corpus, e a instrução no system prompt força ele a usar SÓ esse contexto.

### Pipeline RAG ingênuo vs avançado

**Ingênuo** (o que o 05 implementa):
```
query → retrieve top-k → prompt = "contexto: …\npergunta: …" → LLM → resposta
```

**Avançado** (production-grade):
```
query 
  → query rewriting (LLM reescreve ambiguidades, expande acrônimos)
  → multi-query (gera 3-5 variações da pergunta, busca cada uma)
  → retrieve + rerank (como no 04)
  → reordering (top-1 no início, top-2 no fim — lost in the middle)
  → context compression (LLM pequeno resume cada chunk pra caber mais no contexto)
  → LLM principal gera resposta com citações [1], [2]
  → verifier (segundo LLM checa se a resposta está ancorada no contexto)
```

### Lost in the middle

Paper de Liu et al. (2023) mostrou: LLMs com contextos longos prestam **mais atenção** no começo e no fim do contexto, e **menos** no meio. Então:

- `top_k` grande demais (>10) começa a prejudicar.
- A ordem importa: coloque o doc mais relevante **no início** ou **no fim**, nunca no meio.
- Muitas libs (LangChain, LlamaIndex) têm "reorder" helpers que fazem esse shuffle automaticamente.

### Anti-hallucination: técnicas

1. **System prompt explícito** — "use APENAS os trechos, cite fontes, diga não sei se não encontrar". O 05 faz isso.
2. **Temperatura baixa** — `T=0.1` força determinismo, reduz criatividade-que-vira-hallucination.
3. **Cite-or-die** — rejeitar respostas sem citações no pós-processamento.
4. **Faithfulness scoring** — segundo modelo (ou LLM-judge) pontua se a resposta é ancorada. Frameworks: Ragas, TruLens, DeepEval.
5. **Guardrails/NeMo** — validadores declarativos (regex + LLM) que bloqueiam saída indevida.

### Chunking — o parâmetro mais subestimado

Regras práticas para texto em português:

- **Chunk size**: 300-600 tokens (~200-400 palavras). Cabe uma ideia completa sem diluir.
- **Overlap**: 10-20% — evita que informação seja cortada em fronteiras.
- **Semântico** vs **fixo**: chunker semântico (quebra em `\n\n`, fim de frase, parágrafo) > fixo por tokens. Libs: `RecursiveCharacterTextSplitter` do LangChain, `SemanticSplitterNodeParser` do LlamaIndex.
- **Parent-document retrieval**: indexa chunks pequenos mas entrega o pai (parágrafo inteiro) pro LLM. Melhor de dois mundos.

### RAG × Fine-tuning × Long context

| Problema | Solução |
|---|---|
| LLM precisa conhecer **fatos frescos/privados** | **RAG** |
| LLM precisa **falar um estilo/formato** específico | **Fine-tuning** (LoRA, SFT) |
| LLM precisa entender **um domínio pequeno e fechado** | **Long context** (jogar tudo no prompt) |
| Combinação | RAG + fine-tuning estilístico |

Regra: fine-tuning ensina **comportamento**, RAG injeta **conhecimento**. Misturar funciona.

### Avaliação de RAG (essencial em produção)

- **Context precision** — dos docs retornados, quantos são relevantes pra pergunta?
- **Context recall** — dos docs relevantes no corpus, quantos foram retornados?
- **Faithfulness** — a resposta é ancorada no contexto retornado? (LLM-as-judge)
- **Answer relevance** — a resposta responde à pergunta? (LLM-as-judge)
- **End-to-end accuracy** — em queries com ground-truth, a resposta bate?

Frameworks: **Ragas** (o mais usado), **TruLens**, **DeepEval**. O 05 não tem isso — é um próximo passo natural.

---

## 7. Matriz de decisão — quando usar cada um

### Por tipo de query

| Query típica | Recomendação | Por quê |
|---|---|---|
| Uma palavra ou sigla específica (`SQLite`, `CVE-2024-3094`) | **BM25** (01) | Literal manda |
| Pergunta em linguagem natural (`como isolar um serviço lento?`) | **Hybrid + rerank** (04) | Paráfrase + precisão |
| Conversação longa com o usuário (`me ajuda a decidir entre X e Y`) | **RAG** (05) | Precisa gerar resposta |
| "Achar docs parecidos com este" (given-doc search) | **Vector** (02) | Doc inteiro → embedding |
| Autocomplete de nome | **BM25 prefix** ou trie | Latência manda |
| Cross-lingual (query PT, corpus EN) | **Vector multilingual** (02) | Só embeddings resolvem |

### Por tamanho do corpus

| Corpus | Stack viável |
|---|---|
| <1.000 docs | Tudo em memória, BM25 + vector brute-force, rerank livre |
| 1k–100k | BM25 em memória, vector com HNSW, rerank no top-20 |
| 100k–10M | BM25 com index (Tantivy/Lucene), vector com HNSW persistente, sharding simples |
| 10M–1B | Inverted index distribuído (ES, OpenSearch), ANN particionado (Milvus, Vespa), rerank só em top-50 |
| >1B | Infra especializada (Vespa, Weaviate cloud), sharding + replicação agressiva |

### Por budget de latência

| Budget | Máximo que cabe |
|---|---|
| <20ms | só BM25 em memória |
| <100ms | BM25 + vector (hybrid) |
| <500ms | hybrid + rerank |
| <5s | hybrid + rerank + LLM pequeno (3-7B) |
| <30s | RAG avançado (query rewrite + multi-retrieval + re-rank + LLM grande) |

### Por custo / hardware

| Recurso | Limite |
|---|---|
| Sem GPU | BM25 (01). Embeddings em CPU são 10-50× mais lentas mas viáveis pra corpus <100k. |
| GPU modesta (4-8GB) | 02, 03 funcionam bem. 04 roda com reranker base (250-280M). 05 roda com LLM 3B quantizado. |
| GPU >12GB | Tudo roda, inclusive reranker v2-m3 e LLM 7B. É o que o 04/05 assumem. |
| Sem modelo local, só API | Use Cohere/OpenAI/Voyage embeddings + Cohere rerank + GPT/Claude. Custo linear com volume. |

---

## 8. Algoritmos — cheat sheet final

| Algoritmo | Complexidade | Onde usa | Quando quebra |
|---|---|---|---|
| **BM25** | O(∑posting) ≈ O(k log N) com inverted index | 01, 03, 04, 05 | Sinônimos, multilingual |
| **Bi-encoder (e5)** | O(D) encode + O(log N) HNSW | 02, 03, 04, 05 | Termos literais raros, corpus pequeno |
| **HNSW** | O(log N) empírico, recall >95% | Qdrant em 02-05 | Corpus <1k docs (brute force é melhor) |
| **RRF** | O(k) — só itera listas já ordenadas | 03, 04, 05 | Quando os dois métodos concordam errado |
| **Cross-encoder** | O(K · T²) onde T = tamanho da seq. | 04, 05 | Sem GPU, ou K>50 |
| **LLM geração** | O(tokens_in · tokens_out) | 05 | Context window estourado, custo alto |

---

## 9. Como navegar a progressão

Ordem sugerida pra ler/rodar e fixar os conceitos:

1. Leia **§1 (Glossário)** — referência rápida.
2. `01-keyword-search/` + **§2** — entenda BM25 "na mão".
3. `02-vector-search/` + **§3** — compare a mesma query BM25 × vector.
4. `03-hybrid-search/` + **§4** — toggle entre os 3 modos na UI, descubra onde cada um perde.
5. `04-reranking/` + **§5** — lado a lado hybrid × rerank, veja `↑N` / `↓N`.
6. `05-rag/` + **§6** — pergunte ao LLM, veja as citações e o breakdown de latência.
7. Volta em **§7 e §8** pra fechar a matriz de decisão.

Se for fazer um projeto real, a regra é: **comece pelo BM25**. Sempre. Adicione vector quando o recall de paráfrase virar problema (olhe os logs de queries falhas). Adicione hybrid assim que tiver os dois. Adicione rerank quando a ordem do top-10 importar. Adicione RAG quando o usuário tiver que ler a resposta em vez dos docs.

Pular etapas custa caro — cada camada resolve um problema específico e introduz custo operacional próprio.
