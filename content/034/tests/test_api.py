"""HTTP + WebSocket tests against the FastAPI app."""
from __future__ import annotations

import json
from unittest.mock import AsyncMock

import pytest
from fastapi.testclient import TestClient

from backend.main import create_app


@pytest.fixture
def client(monkeypatch):
    app = create_app()

    async def fake_chat(history, prompt):
        return f"echo:{prompt}"

    async def fake_stream(history, prompt):
        for t in ["a", "b", "c"]:
            yield t

    async def fake_health():
        return {"ok": True, "models": ["test"]}

    with TestClient(app) as c:
        c.app.state.llm.chat = fake_chat
        c.app.state.llm.chat_stream = fake_stream
        c.app.state.llm.health = fake_health
        yield c


def test_config_endpoint(client):
    r = client.get("/api/config")
    assert r.status_code == 200
    data = r.json()
    assert "ui" in data and "voice" in data
    assert data["ui"]["title"]


def test_health_endpoint(client):
    r = client.get("/api/health")
    assert r.status_code == 200
    data = r.json()
    assert data["status"] == "ONLINE"
    assert data["llm"]["ok"] is True


def test_chat_endpoint(client):
    r = client.post("/api/chat", json={"prompt": "ping", "history": []})
    assert r.status_code == 200
    assert r.json()["reply"] == "echo:ping"


def test_frontend_root(client):
    r = client.get("/")
    assert r.status_code == 200
    assert "RED QUEEN" in r.text or "redqueen" in r.text.lower()


def test_websocket_chat_flow(client):
    with client.websocket_connect("/ws") as ws:
        hello = ws.receive_json()
        assert hello["type"] == "ready"

        ws.send_text(json.dumps({"type": "ping"}))
        assert ws.receive_json()["type"] == "pong"

        ws.send_text(json.dumps({"type": "chat", "prompt": "hello"}))
        msgs = []
        while True:
            m = ws.receive_json()
            msgs.append(m)
            if m["type"] == "end":
                break
        types = [m["type"] for m in msgs]
        assert types[0] == "start"
        assert "token" in types
        assert msgs[-1]["reply"] == "abc"


def test_websocket_reset(client):
    with client.websocket_connect("/ws") as ws:
        ws.receive_json()
        ws.send_text(json.dumps({"type": "reset"}))
        assert ws.receive_json() == {"type": "reset_ok"}


def test_websocket_bad_json(client):
    with client.websocket_connect("/ws") as ws:
        ws.receive_json()
        ws.send_text("not-json")
        m = ws.receive_json()
        assert m["type"] == "error" and m["error"] == "bad_json"
