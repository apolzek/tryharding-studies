"""End-to-end RAG tests — verify the LLM answers only from retrieved context."""
from __future__ import annotations

import os

import httpx

BASE = os.environ.get("BASE_URL", "http://localhost:8000")


def test_health() -> None:
    r = httpx.get(f"{BASE}/health", timeout=30)
    body = r.json()
    assert body["status"] == "ok"
    assert "qwen" in body["llm"].lower() or "llama" in body["llm"].lower()


def test_ask_returns_answer_and_sources() -> None:
    r = httpx.get(f"{BASE}/ask", params={"q": "qual banco para detectar fraude em grafos?"}, timeout=180)
    assert r.status_code == 200
    data = r.json()
    assert data["answer"].strip()
    assert len(data["sources"]) >= 1
    assert data["ms_llm"] > 0


def test_top_source_is_neo4j_for_graph_fraud() -> None:
    r = httpx.get(f"{BASE}/ask", params={"q": "qual banco para detectar fraude em grafos?"}, timeout=180).json()
    titles = [s["title"] for s in r["sources"]]
    assert "Neo4j" in titles[:3], f"expected Neo4j in sources, got {titles}"


def test_answer_cites_sources_with_brackets() -> None:
    """The system prompt asks the LLM to cite sources like [1], [2]."""
    r = httpx.get(
        f"{BASE}/ask",
        params={"q": "qual banco para detectar fraude em grafos?"},
        timeout=180,
    ).json()
    assert "[" in r["answer"], f"no bracket citation in: {r['answer']}"


def test_ask_out_of_domain_does_not_hallucinate() -> None:
    """Question totally off-topic should result in a refusal, not invented facts."""
    r = httpx.get(
        f"{BASE}/ask",
        params={"q": "qual a receita de brigadeiro?"},
        timeout=180,
    ).json()
    low = r["answer"].lower()
    refusal_words = [
        "não sei", "não sabe", "não encontro", "não está", "não há",
        "sem informa", "não posso", "não é possível", "fora do",
    ]
    assert any(w in low for w in refusal_words), f"expected refusal, got: {r['answer']}"


def test_observability_question_mentions_prometheus() -> None:
    r = httpx.get(
        f"{BASE}/ask",
        params={"q": "como coleto métricas de microsserviços?"},
        timeout=180,
    ).json()
    assert "Prometheus".lower() in r["answer"].lower() or "prometheus" in [s["title"].lower() for s in r["sources"]]
