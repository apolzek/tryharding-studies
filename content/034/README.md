---
title: RedQueen — Jarvis × Resident Evil
tags: [ai-ml, ollama, fastapi, python, discord, frontend]
status: stable
---

# RedQueen — Jarvis × Resident Evil

Hostile-but-loyal AI terminal. Voice input, Discord bridge, local Llama,
ASCII-driven hacker UI with matrix rain, glitch text, scanlines and CRT flicker.

```
╔════════════════════════════════════════════════╗
║  R E D   Q U E E N  //  HIVE MAINFRAME v3.14   ║
╚════════════════════════════════════════════════╝
```

## Stack

- **Backend**: FastAPI + WebSocket streaming, pydantic config
- **LLM**: Ollama (local Llama 3.2 by default, swap via YAML)
- **Voice**: Browser `SpeechRecognition` (STT) + `speechSynthesis` (TTS), low-pitch sinister voice
- **Discord**: `discord.py` with voice-channel join (`!rq join`, `!rq ask ...`, `!rq leave`, `!rq reset`)
- **Frontend**: vanilla HTML/CSS/JS — matrix rain canvas, glitch-clipped red text, scanlines, CRT flicker, boot sequence
- **Config**: every integration lives in `config.yaml`

## Layout

```
034/
├── config.yaml          # all integrations
├── .env.example         # secrets
├── requirements.txt
├── Makefile
├── run.py
├── backend/
│   ├── main.py          # FastAPI app + WS + static mount
│   ├── config.py        # YAML loader + env overrides
│   ├── llm.py           # Ollama streaming client
│   └── discord_bot.py   # Discord text + voice
├── frontend/
│   ├── index.html
│   ├── style.css        # red/crimson hacker theme
│   ├── app.js           # WS + STT/TTS + matrix + boot
│   └── ascii.js         # RedQueen art + boot log
└── tests/               # pytest (config, llm mock, api, ws, frontend assets)
```

## Quick start

```bash
# 1. install
make install

# 2. pull a local model (once)
ollama pull llama3.2

# 3. launch
make run
# open http://localhost:8000
```

### Discord

1. Create an app + bot in the Discord developer portal, enable *Message Content* and *Server Members* intents.
2. Put the token in `.env` (or export `REDQUEEN_DISCORD_TOKEN`).
3. Set `discord.enabled: true` in `config.yaml`.
4. Invite the bot; in a text channel:
   - `!rq ask how do I disable the laser grid?`
   - `!rq join` (while you are in a voice channel)
   - `!rq leave`
   - `!rq reset`

### Environment overrides

| Variable | Effect |
|---|---|
| `REDQUEEN_CONFIG` | Path to a different YAML |
| `REDQUEEN_OLLAMA_URL` | Override `llm.ollama.base_url` |
| `REDQUEEN_OLLAMA_MODEL` | Override `llm.ollama.model` |
| `REDQUEEN_DISCORD_TOKEN` | Injected into `discord.token` |

## UI controls

| Input | Action |
|---|---|
| `Space` (outside input) | Toggle mic |
| `Esc` | Stop TTS |
| `/reset` | Purge conversation memory |
| `/stop` | Halt speech |
| `/clear` | Clear console |

## Tests

```bash
make test
```

Covers config loader, Ollama client (mocked with `httpx.MockTransport`), HTTP
endpoints, WebSocket streaming/reset/error paths, and frontend asset wiring.

## Notes

- The frontend does *not* require a Discord connection — it speaks to the backend WS directly.
- TTS picks a low-pitch voice by default (`tts_pitch: 0.6`). Override `voice.browser.tts_voice` with an exact voice name from `speechSynthesis.getVoices()`.
- `llm.system_prompt` is the entire persona — edit it to shift the character.
