"""Integration tests hitting the running backend over HTTP."""
from __future__ import annotations

import os

import httpx
import pytest

BASE = os.environ.get("BASE_URL", "http://localhost:8000")


def test_health() -> None:
    r = httpx.get(f"{BASE}/health", timeout=10)
    assert r.status_code == 200
    body = r.json()
    assert body["status"] == "ok"
    assert int(body["docs"]) >= 30


def test_search_relevance_graph_db() -> None:
    r = httpx.get(f"{BASE}/search", params={"q": "banco de grafos para detecção de fraude"}, timeout=10)
    assert r.status_code == 200
    hits = r.json()["hits"]
    assert hits, "expected at least one hit"
    assert hits[0]["title"] == "Neo4j"


def test_search_relevance_containers() -> None:
    r = httpx.get(f"{BASE}/search", params={"q": "orquestração de containers"}, timeout=10)
    assert r.status_code == 200
    titles = [h["title"] for h in r.json()["hits"]]
    assert "Kubernetes" in titles[:2]


def test_search_score_descending() -> None:
    r = httpx.get(f"{BASE}/search", params={"q": "linguagem compilada goroutines"}, timeout=10)
    hits = r.json()["hits"]
    scores = [h["score"] for h in hits]
    assert scores == sorted(scores, reverse=True)


def test_empty_query_rejected() -> None:
    r = httpx.get(f"{BASE}/search", params={"q": ""}, timeout=10)
    assert r.status_code == 422


def test_stopword_only_returns_empty() -> None:
    r = httpx.get(f"{BASE}/search", params={"q": "a e o que"}, timeout=10)
    assert r.status_code == 200
    assert r.json()["hits"] == []


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
