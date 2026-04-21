"""Reranker tests — verify the cross-encoder refines the hybrid ranking."""
from __future__ import annotations

import os

import httpx

BASE = os.environ.get("BASE_URL", "http://localhost:8000")


def test_health() -> None:
    r = httpx.get(f"{BASE}/health", timeout=30)
    body = r.json()
    assert body["status"] == "ok"
    assert int(body["points"]) >= 30
    assert "bge-reranker" in body["reranker"]


def test_rerank_never_drops_top_hit_completely() -> None:
    """Reranker should keep the top hybrid candidate in the top-k."""
    q = "banco de dados para séries temporais"
    hyb = httpx.get(f"{BASE}/search", params={"q": q, "mode": "hybrid", "k": 5}, timeout=60).json()
    rer = httpx.get(f"{BASE}/search", params={"q": q, "mode": "rerank", "k": 5}, timeout=60).json()
    assert hyb["hits"][0]["title"] in [h["title"] for h in rer["hits"]]


def test_rerank_score_orders_descending() -> None:
    r = httpx.get(f"{BASE}/search", params={"q": "linguagem segura para sistemas", "mode": "rerank"}, timeout=60).json()
    scores = [h["rerank_score"] for h in r["hits"]]
    assert all(a >= b for a, b in zip(scores, scores[1:])), f"non-monotone: {scores}"


def test_rerank_exposes_orig_hybrid_rank() -> None:
    r = httpx.get(f"{BASE}/search", params={"q": "orquestração de containers", "mode": "rerank"}, timeout=60).json()
    for h in r["hits"]:
        assert h["rrf_rank"] is not None


def test_rerank_puts_react_for_ui_components_query() -> None:
    """Paraphrase that hybrid might mix with other frontend concepts."""
    q = "biblioteca que renderiza componentes no navegador"
    r = httpx.get(f"{BASE}/search", params={"q": q, "mode": "rerank", "k": 5}, timeout=60).json()
    titles = [h["title"] for h in r["hits"]]
    assert "React" in titles[:3], f"expected React in top-3 after rerank, got {titles}"


def test_rerank_latency_tolerable() -> None:
    r = httpx.get(f"{BASE}/search", params={"q": "monitoramento com prometheus", "mode": "rerank"}, timeout=60).json()
    assert r["ms"] < 5000, f"rerank too slow: {r['ms']}ms"
