# POC 010 — Discord Bot + Whisper API (Transcrição Local)

Bot Discord que monitora o canal `robo`, transcreve mensagens de voz usando Whisper rodando localmente na GPU, e grava mensagens de texto e áudios de canal de voz.

## Arquitetura

```
Discord
  │
  ▼
discord-bot          ──► whisper-api (faster-whisper + CUDA)
  │  recebe audio/ogg       transcreve com large-v3
  │  responde no canal  ◄──  retorna texto
  │
  ├── messages.log     (texto + transcrições)
  └── recordings/      (áudios do canal de voz .pcm)
```

## Pré-requisitos

- Docker + Docker Compose
- NVIDIA GPU com driver >= 525 e [NVIDIA Container Toolkit](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/install-guide.html)
- Bot Discord criado no [Developer Portal](https://discord.com/developers/applications) com as seguintes permissões: `Read Messages`, `Connect`, `Speak`, `View Channel`
- Intents habilitados no portal: `SERVER MEMBERS INTENT`, `MESSAGE CONTENT INTENT`

## Configuração

Edite `discord-bot/.env` com o token do bot:

```env
DISCORD_TOKEN=seu_token_aqui
APPLICATION_ID=seu_application_id_aqui
```

## Rodando com Docker

```bash
cd pocs/010
docker compose up --build
```

O `whisper-api` baixa o modelo `large-v3` (~3 GB) na primeira execução e armazena em volume. Nas execuções seguintes sobe em segundos.

O `discord-bot` aguarda o health check da API passar antes de iniciar.

## Rodando sem Docker (desenvolvimento)

**Terminal 1 — Whisper API:**
```bash
cd whisper-api
./start.sh
```

**Terminal 2 — Bot:**
```bash
cd discord-bot
npm install
node index.js
```

## O que o bot faz

| Evento | Ação |
|---|---|
| Mensagem de texto no canal `robo` | Salva em `messages.log` |
| Arquivo de áudio (`audio/*`) no canal `robo` | Transcreve via Whisper e responde no canal |
| Usuário entra no canal de voz `robo` | Bot entra e grava o áudio em `.pcm` |
| Canal de voz fica vazio | Bot sai automaticamente |

## Whisper API

Serviço HTTP independente que expõe o modelo [faster-whisper](https://github.com/SYSTRAN/faster-whisper) via REST.

### Endpoints

#### `GET /health`

Verifica se o serviço está operacional.

**Resposta:**
```json
{
  "status": "ok",
  "model": "large-v3",
  "device": "cuda"
}
```

---

#### `POST /transcribe`

Transcreve um arquivo de áudio para texto.

**Request:** `multipart/form-data`

| Campo | Tipo | Obrigatório | Descrição |
|---|---|---|---|
| `file` | arquivo | sim | Áudio a transcrever (ogg, mp3, wav, mp4, webm, flac) |
| `language` | string | não | Código do idioma (ex: `pt`, `en`). Omitir para detecção automática |

**Exemplo com curl:**
```bash
curl -X POST http://localhost:8001/transcribe \
  -F "file=@audio.ogg" \
  -F "language=pt"
```

**Resposta:**
```json
{
  "text": "Robo, você está ligado?",
  "language": "pt",
  "language_probability": 1.0
}
```

**Erros:**

| Código | Motivo |
|---|---|
| `500` | Falha na transcrição (detalhe no campo `detail`) |

### Variáveis de ambiente

| Variável | Padrão | Descrição |
|---|---|---|
| `WHISPER_MODEL` | `large-v3` | Tamanho do modelo: `tiny`, `base`, `small`, `medium`, `large-v3` |
| `WHISPER_DEVICE` | `cuda` | `cuda` para GPU, `cpu` para CPU |
| `WHISPER_COMPUTE` | `float16` | `float16` (GPU), `int8` (CPU ou GPU com menos VRAM) |

### Modelos disponíveis

| Modelo | VRAM | Velocidade | Qualidade |
|---|---|---|---|
| `tiny` | ~1 GB | muito rápido | baixa |
| `base` | ~1 GB | rápido | razoável |
| `small` | ~2 GB | rápido | boa |
| `medium` | ~5 GB | médio | muito boa |
| `large-v3` | ~6 GB | médio | excelente |

## Estrutura de arquivos

```
pocs/010/
├── docker-compose.yml
├── discord-bot/
│   ├── Dockerfile
│   ├── index.js
│   ├── package.json
│   ├── .env
│   ├── messages.log        # log de textos e transcrições
│   └── recordings/         # áudios do canal de voz (.pcm)
└── whisper-api/
    ├── Dockerfile
    ├── server.py
    ├── requirements.txt
    └── start.sh            # script para rodar sem Docker
```
