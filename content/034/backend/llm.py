"""Ollama client for RedQueen. Streams tokens over async generator."""
from __future__ import annotations

import json
from collections.abc import AsyncIterator
from typing import Any

import httpx

from .config import LLMCfg


class OllamaClient:
    def __init__(self, cfg: LLMCfg):
        self.cfg = cfg
        self.base = cfg.ollama.base_url.rstrip("/")

    async def health(self) -> dict[str, Any]:
        try:
            async with httpx.AsyncClient(timeout=3.0) as c:
                r = await c.get(f"{self.base}/api/tags")
                r.raise_for_status()
                return {"ok": True, "models": [m["name"] for m in r.json().get("models", [])]}
        except Exception as exc:
            return {"ok": False, "error": str(exc)}

    def _payload(self, history: list[dict[str, str]], prompt: str, stream: bool) -> dict[str, Any]:
        messages = [{"role": "system", "content": self.cfg.system_prompt}]
        messages.extend(history[-self.cfg.max_history :])
        messages.append({"role": "user", "content": prompt})
        return {
            "model": self.cfg.ollama.model,
            "messages": messages,
            "stream": stream,
            "keep_alive": self.cfg.ollama.keep_alive,
            "options": {"temperature": self.cfg.temperature},
        }

    async def chat_stream(
        self, history: list[dict[str, str]], prompt: str
    ) -> AsyncIterator[str]:
        payload = self._payload(history, prompt, stream=True)
        url = f"{self.base}/api/chat"
        async with httpx.AsyncClient(timeout=self.cfg.ollama.timeout) as c:
            async with c.stream("POST", url, json=payload) as r:
                r.raise_for_status()
                async for line in r.aiter_lines():
                    if not line.strip():
                        continue
                    try:
                        chunk = json.loads(line)
                    except json.JSONDecodeError:
                        continue
                    msg = chunk.get("message", {})
                    token = msg.get("content") or ""
                    if token:
                        yield token
                    if chunk.get("done"):
                        return

    async def chat(self, history: list[dict[str, str]], prompt: str) -> str:
        payload = self._payload(history, prompt, stream=False)
        async with httpx.AsyncClient(timeout=self.cfg.ollama.timeout) as c:
            r = await c.post(f"{self.base}/api/chat", json=payload)
            r.raise_for_status()
            return r.json().get("message", {}).get("content", "")
