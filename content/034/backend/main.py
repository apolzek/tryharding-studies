"""RedQueen FastAPI app — HTTP, WebSocket, static frontend."""
from __future__ import annotations

import asyncio
import json
import logging
from contextlib import asynccontextmanager
from pathlib import Path

from fastapi import FastAPI, WebSocket, WebSocketDisconnect
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import FileResponse, JSONResponse
from fastapi.staticfiles import StaticFiles

from .config import Config, get_config
from .llm import OllamaClient

log = logging.getLogger("redqueen")

FRONTEND_DIR = Path(__file__).resolve().parent.parent / "frontend"


@asynccontextmanager
async def lifespan(app: FastAPI):
    cfg: Config = get_config()
    app.state.cfg = cfg
    app.state.llm = OllamaClient(cfg.llm)
    app.state.discord_task = None
    if cfg.discord.enabled and cfg.discord.token:
        try:
            from .discord_bot import start_discord  # lazy import

            app.state.discord_task = asyncio.create_task(start_discord(cfg, app.state.llm))
            log.info("Discord bot launched")
        except Exception as exc:  # noqa: BLE001
            log.exception("Discord bot failed to start: %s", exc)
    yield
    if app.state.discord_task:
        app.state.discord_task.cancel()


def create_app() -> FastAPI:
    cfg = get_config()
    app = FastAPI(title="RedQueen", lifespan=lifespan)
    app.add_middleware(
        CORSMiddleware,
        allow_origins=cfg.server.cors_origins,
        allow_methods=["*"],
        allow_headers=["*"],
    )

    @app.get("/api/config")
    async def api_config():
        c = app.state.cfg
        return {
            "ui": c.ui.model_dump(),
            "voice": c.voice.model_dump(),
            "llm_model": c.llm.ollama.model,
            "discord_enabled": c.discord.enabled,
        }

    @app.get("/api/health")
    async def api_health():
        llm_health = await app.state.llm.health()
        return {
            "status": "ONLINE" if llm_health["ok"] else "DEGRADED",
            "llm": llm_health,
            "discord": app.state.cfg.discord.enabled,
        }

    @app.post("/api/chat")
    async def api_chat(payload: dict):
        prompt = payload.get("prompt", "")
        history = payload.get("history", [])
        reply = await app.state.llm.chat(history, prompt)
        return {"reply": reply}

    @app.websocket("/ws")
    async def ws_endpoint(ws: WebSocket):
        await ws.accept()
        history: list[dict[str, str]] = []
        await ws.send_json({"type": "ready", "model": app.state.cfg.llm.ollama.model})
        try:
            while True:
                raw = await ws.receive_text()
                try:
                    data = json.loads(raw)
                except json.JSONDecodeError:
                    await ws.send_json({"type": "error", "error": "bad_json"})
                    continue

                kind = data.get("type", "chat")
                if kind == "reset":
                    history.clear()
                    await ws.send_json({"type": "reset_ok"})
                    continue
                if kind == "ping":
                    await ws.send_json({"type": "pong"})
                    continue

                prompt = (data.get("prompt") or "").strip()
                if not prompt:
                    continue

                await ws.send_json({"type": "start"})
                full = []
                try:
                    async for token in app.state.llm.chat_stream(history, prompt):
                        full.append(token)
                        await ws.send_json({"type": "token", "token": token})
                except Exception as exc:  # noqa: BLE001
                    await ws.send_json({"type": "error", "error": str(exc)})
                    continue

                reply = "".join(full)
                history.append({"role": "user", "content": prompt})
                history.append({"role": "assistant", "content": reply})
                await ws.send_json({"type": "end", "reply": reply})
        except WebSocketDisconnect:
            return

    if FRONTEND_DIR.exists():
        app.mount("/static", StaticFiles(directory=FRONTEND_DIR), name="static")

        @app.get("/")
        async def root():
            index = FRONTEND_DIR / "index.html"
            if index.exists():
                return FileResponse(index)
            return JSONResponse({"error": "frontend missing"}, status_code=404)

    return app


app = create_app()
