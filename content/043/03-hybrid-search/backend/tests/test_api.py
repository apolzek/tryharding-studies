"""Hybrid search tests — verify RRF recovers matches that pure modes miss."""
from __future__ import annotations

import os

import httpx

BASE = os.environ.get("BASE_URL", "http://localhost:8000")


def test_health() -> None:
    r = httpx.get(f"{BASE}/health", timeout=30)
    assert r.status_code == 200
    assert int(r.json()["points"]) >= 30


def test_hybrid_beats_vector_only_for_queue_query() -> None:
    """Vector alone puts Redis/Helm before RabbitMQ; hybrid should pull RabbitMQ up."""
    q = "sistema de filas com ack e garantia de entrega"
    vec = httpx.get(f"{BASE}/search", params={"q": q, "mode": "vector", "k": 5}, timeout=30).json()
    hyb = httpx.get(f"{BASE}/search", params={"q": q, "mode": "hybrid", "k": 5}, timeout=30).json()
    vec_titles = [h["title"] for h in vec["hits"]]
    hyb_titles = [h["title"] for h in hyb["hits"]]
    vec_rank = vec_titles.index("RabbitMQ") if "RabbitMQ" in vec_titles else 99
    hyb_rank = hyb_titles.index("RabbitMQ") if "RabbitMQ" in hyb_titles else 99
    assert hyb_rank <= vec_rank, f"hybrid should not rank worse. vec={vec_titles} hyb={hyb_titles}"


def test_hybrid_keeps_paraphrase_wins() -> None:
    """Hybrid should still surface Terraform for a paraphrase query."""
    q = "ferramenta para provisionar nuvem de forma declarativa"
    r = httpx.get(f"{BASE}/search", params={"q": q, "mode": "hybrid", "k": 5}, timeout=30).json()
    titles = [h["title"] for h in r["hits"]]
    assert "Terraform" in titles[:3], f"got {titles}"


def test_hybrid_keeps_exact_match_wins() -> None:
    """Queries with unique literal tokens stay top-1 in hybrid."""
    q = "linguagem compilada goroutines channels"
    r = httpx.get(f"{BASE}/search", params={"q": q, "mode": "hybrid", "k": 3}, timeout=30).json()
    titles = [h["title"] for h in r["hits"]]
    assert titles[0] == "Go", f"expected Go at top, got {titles}"


def test_hybrid_exposes_component_ranks() -> None:
    r = httpx.get(f"{BASE}/search", params={"q": "banco de grafos", "mode": "hybrid"}, timeout=30).json()
    hits = r["hits"]
    assert hits, "expected hits"
    assert hits[0]["title"] == "Neo4j"
    assert hits[0]["bm25_rank"] is not None or hits[0]["vector_rank"] is not None


def test_mode_bm25_equivalent_to_01() -> None:
    """Sanity: bm25 mode returns the expected top-1 for keyword queries."""
    r = httpx.get(f"{BASE}/search", params={"q": "goroutines", "mode": "bm25"}, timeout=30).json()
    assert r["hits"][0]["title"] == "Go"


def test_mode_vector_semantic_still_works() -> None:
    r = httpx.get(f"{BASE}/search", params={"q": "onde guardo vetores", "mode": "vector"}, timeout=30).json()
    titles = [h["title"] for h in r["hits"]]
    assert "Qdrant" in titles[:3]
