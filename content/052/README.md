# 052 — Brasilzão

Portal para brasileiros: dica do dia, direitos, impostos, políticos e história.

## O que tem

- **Dica do dia** — 30+ dicas úteis rotacionadas por data (direitos do consumidor, trânsito, trabalhista, saúde, finanças, impostos, etc.).
- **Direitos & Normas** — 9 áreas (CDC, CTB/Detran, CLT, SUS, INSS, moradia, documentos, penal, digital/LGPD) com artigos, prazos e fontes.
- **Impostos** — Federais (IRPF, IR investimentos, IPI, PIS/COFINS, CSLL, INSS), estaduais (ICMS, IPVA, ITCMD), municipais (IPTU, ISS, ITBI) + calendário tributário.
- **Políticos** — **dados em tempo real** via APIs oficiais da [Câmara dos Deputados](https://dadosabertos.camara.leg.br/) e do [Senado Federal](https://www12.senado.leg.br/dados-abertos). Inclui dashboard por partido (distribuição nacional + por UF) e ficha de cada legenda com fotos.
- **História do Brasil** — linha do tempo de pré-colonial até a Nova República, com 80+ eventos datados.

## Stack

- **Backend**: Node 20 + Express (ESM), cache em memória (TTL 6h) para APIs externas.
- **Frontend**: React 18 + Vite + React Router. Sem bibliotecas de UI: CSS próprio com cores da bandeira.
- **Orquestração**: Docker Compose.

## Como rodar

```bash
docker compose up --build
```

Depois:
- Frontend: http://localhost:5173
- Backend: http://localhost:4000
- Health: http://localhost:4000/health

Para rodar sem Docker:

```bash
cd backend && npm install && npm run dev   # porta 4000
cd frontend && npm install && npm run dev  # porta 5173 (proxy pra 4000)
```

## Endpoints

```
GET /api/tips/today                    dica do dia (rotação diária determinística)
GET /api/tips                          lista completa
GET /api/rights                        categorias de direitos
GET /api/rights/:id                    itens da categoria (consumidor, transito, trabalho, ...)
GET /api/taxes                         todos os impostos + calendário
GET /api/history                       períodos históricos com eventos
GET /api/politicians/deputados         lista da Câmara (cacheada)
GET /api/politicians/senadores         lista do Senado (cacheada)
GET /api/politicians/partidos          agregação por partido (Câmara + Senado)
GET /api/politicians/partidos/:sigla   detalhe do partido com distribuição por UF
```

## Fontes dos dados dinâmicos

- Câmara — `https://dadosabertos.camara.leg.br/api/v2/deputados`
- Senado — `https://legis.senado.leg.br/dadosabertos/senador/lista/atual`

Conteúdo estático (direitos, impostos, história, dicas) é curado em JSON e pode ser ampliado editando os arquivos em `backend/src/data/`.

## Aviso

Conteúdo educativo. Não substitui consulta jurídica, contábil ou médica formal. Verifique sempre a fonte citada.
