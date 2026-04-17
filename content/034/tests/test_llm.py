"""LLM client tests using httpx MockTransport."""
from __future__ import annotations

import json

import httpx
import pytest

from backend import llm as llm_mod
from backend.config import LLMCfg, OllamaCfg
from backend.llm import OllamaClient


def _cfg() -> LLMCfg:
    return LLMCfg(ollama=OllamaCfg(base_url="http://mock:11434", model="test"))


class _ClientFactory:
    """Drop-in replacement for httpx.AsyncClient that injects a MockTransport."""

    def __init__(self, handler):
        self._handler = handler

    def __call__(self, *args, **kwargs):
        kwargs["transport"] = httpx.MockTransport(self._handler)
        # use real client class — bypass the monkeypatch
        return _RealAsyncClient(*args, **kwargs)


_RealAsyncClient = httpx.AsyncClient


@pytest.fixture
def patch_httpx(monkeypatch):
    def install(handler):
        monkeypatch.setattr(llm_mod.httpx, "AsyncClient", _ClientFactory(handler))
    return install


@pytest.mark.asyncio
async def test_health_ok(patch_httpx):
    patch_httpx(lambda req: httpx.Response(200, json={"models": [{"name": "test"}]}))
    h = await OllamaClient(_cfg()).health()
    assert h["ok"] is True
    assert "test" in h["models"]


@pytest.mark.asyncio
async def test_health_fail(patch_httpx):
    def boom(req):
        raise httpx.ConnectError("down")
    patch_httpx(boom)
    h = await OllamaClient(_cfg()).health()
    assert h["ok"] is False
    assert "down" in h["error"]


@pytest.mark.asyncio
async def test_chat_non_stream(patch_httpx):
    def handler(req: httpx.Request) -> httpx.Response:
        body = json.loads(req.content)
        assert body["model"] == "test"
        assert body["stream"] is False
        assert body["messages"][-1]["content"] == "hello"
        return httpx.Response(
            200, json={"message": {"role": "assistant", "content": "welcome, operator"}}
        )
    patch_httpx(handler)
    out = await OllamaClient(_cfg()).chat([], "hello")
    assert out == "welcome, operator"


@pytest.mark.asyncio
async def test_chat_stream_tokens(patch_httpx):
    tokens = ["wel", "come", ", ", "operator"]

    def handler(req: httpx.Request) -> httpx.Response:
        lines = [json.dumps({"message": {"content": t}, "done": False}) for t in tokens]
        lines.append(json.dumps({"message": {"content": ""}, "done": True}))
        return httpx.Response(200, content=("\n".join(lines) + "\n").encode())

    patch_httpx(handler)
    collected = []
    async for tok in OllamaClient(_cfg()).chat_stream([], "hello"):
        collected.append(tok)
    assert "".join(collected) == "welcome, operator"


@pytest.mark.asyncio
async def test_chat_stream_stops_on_done(patch_httpx):
    def handler(req):
        body = json.dumps({"message": {"content": "solo"}, "done": True}) + "\n"
        return httpx.Response(200, content=body.encode())
    patch_httpx(handler)
    out = [t async for t in OllamaClient(_cfg()).chat_stream([], "hi")]
    assert out == ["solo"]


@pytest.mark.asyncio
async def test_chat_stream_skips_bad_lines(patch_httpx):
    def handler(req):
        lines = [
            "garbage-not-json",
            "",
            json.dumps({"message": {"content": "hi"}, "done": False}),
            json.dumps({"message": {"content": ""}, "done": True}),
        ]
        return httpx.Response(200, content=("\n".join(lines) + "\n").encode())
    patch_httpx(handler)
    out = [t async for t in OllamaClient(_cfg()).chat_stream([], "x")]
    assert out == ["hi"]
