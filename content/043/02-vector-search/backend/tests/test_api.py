"""Integration tests for the dense vector search API."""
from __future__ import annotations

import os

import httpx

BASE = os.environ.get("BASE_URL", "http://localhost:8000")


def test_health() -> None:
    r = httpx.get(f"{BASE}/health", timeout=30)
    assert r.status_code == 200
    body = r.json()
    assert body["status"] == "ok"
    assert int(body["points"]) >= 30


def test_semantic_embeddings_query_returns_qdrant() -> None:
    """Query uses words that DON'T appear in the Qdrant doc.

    The doc says 'armazena embeddings de alta dimensão...' — keyword BM25
    would catch the word 'embeddings', but also many other docs. Vector
    search should put Qdrant in the top results via semantic similarity.
    """
    q = "onde eu guardo vetores numéricos para buscar similaridade"
    r = httpx.get(f"{BASE}/search", params={"q": q}, timeout=30)
    assert r.status_code == 200
    titles = [h["title"] for h in r.json()["hits"]]
    assert "Qdrant" in titles[:3], f"expected Qdrant in top-3, got {titles}"


def test_paraphrase_iac() -> None:
    """Query never uses the word 'Terraform', yet should surface it."""
    q = "ferramenta para provisionar nuvem de forma declarativa"
    r = httpx.get(f"{BASE}/search", params={"q": q}, timeout=30)
    titles = [h["title"] for h in r.json()["hits"]]
    assert "Terraform" in titles[:3], f"expected Terraform in top-3, got {titles}"


def test_paraphrase_queue() -> None:
    """'sistema de filas com ack' → should surface RabbitMQ.

    Note: vector search alone often puts Redis/Kafka ahead for this query
    since they share semantic context (messaging/brokers). This is exactly
    the gap that hybrid search (03) closes — BM25 nails the literal word
    'AMQP'/'filas' that only appears in the RabbitMQ doc.
    """
    q = "sistema de filas com ack e garantia de entrega"
    r = httpx.get(f"{BASE}/search", params={"q": q}, timeout=30)
    titles = [h["title"] for h in r.json()["hits"]]
    assert "RabbitMQ" in titles[:5], f"expected RabbitMQ in top-5, got {titles}"


def test_scores_are_cosine_bounded() -> None:
    r = httpx.get(f"{BASE}/search", params={"q": "banco de dados"}, timeout=30)
    for h in r.json()["hits"]:
        assert -1.0 <= h["score"] <= 1.0
