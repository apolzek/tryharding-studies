"""Hybrid search = BM25 (sparse) + Qdrant cosine (dense), fused via RRF."""
from __future__ import annotations

import json
import os
import re
import time
from pathlib import Path

import torch
from fastapi import FastAPI, Query
from fastapi.middleware.cors import CORSMiddleware
from fastapi.staticfiles import StaticFiles
from pydantic import BaseModel
from qdrant_client import QdrantClient
from qdrant_client.http import models as qm
from rank_bm25 import BM25Okapi
from sentence_transformers import SentenceTransformer
from unidecode import unidecode

DOCS_PATH = Path(__file__).parent / "docs.json"
FRONTEND_DIR = Path(__file__).parent / "frontend"
QDRANT_URL = os.environ.get("QDRANT_URL", "http://qdrant:6333")
COLLECTION = "docs"
MODEL_NAME = os.environ.get("EMBED_MODEL", "intfloat/multilingual-e5-small")
DEVICE = "cuda" if torch.cuda.is_available() else "cpu"
RRF_K = 60  # classic RRF constant from the Cormack/Clarke/Buettcher paper

PT_STOPWORDS = {
    "a", "o", "as", "os", "um", "uma", "uns", "umas", "de", "do", "da", "dos", "das",
    "em", "no", "na", "nos", "nas", "por", "para", "com", "sem", "que", "e", "ou",
    "mas", "se", "como", "ao", "aos", "à", "às", "é", "foi", "ser", "são", "este",
    "esta", "isso", "isto", "seu", "sua", "seus", "suas", "sobre", "entre", "também",
}
TOKEN_RE = re.compile(r"[a-z0-9]+")


def tokenize(text: str) -> list[str]:
    text = unidecode(text.lower())
    return [t for t in TOKEN_RE.findall(text) if t not in PT_STOPWORDS and len(t) > 1]


class Doc(BaseModel):
    id: int
    title: str
    content: str


class SearchHit(BaseModel):
    id: int
    title: str
    content: str
    score: float
    bm25_rank: int | None = None
    vector_rank: int | None = None


class SearchResponse(BaseModel):
    query: str
    hits: list[SearchHit]
    ms: float
    mode: str


def load_docs() -> list[Doc]:
    with DOCS_PATH.open(encoding="utf-8") as f:
        return [Doc(**d) for d in json.load(f)]


def wait_for_qdrant(client: QdrantClient, attempts: int = 60) -> None:
    for i in range(attempts):
        try:
            client.get_collections()
            return
        except Exception as e:  # noqa: BLE001
            print(f"[boot] waiting qdrant ({i+1}/{attempts}): {e}")
            time.sleep(1)
    raise RuntimeError("qdrant never became ready")


def ingest(model: SentenceTransformer, client: QdrantClient, docs: list[Doc]) -> None:
    if client.collection_exists(COLLECTION):
        info = client.get_collection(COLLECTION)
        if info.points_count and info.points_count >= len(docs):
            print(f"[boot] collection already populated ({info.points_count} pts)")
            return
        client.delete_collection(COLLECTION)

    dim = model.get_sentence_embedding_dimension()
    client.create_collection(
        collection_name=COLLECTION,
        vectors_config=qm.VectorParams(size=dim, distance=qm.Distance.COSINE),
    )
    texts = [f"passage: {d.title}. {d.content}" for d in docs]
    vectors = model.encode(
        texts,
        batch_size=16,
        normalize_embeddings=True,
        show_progress_bar=False,
        convert_to_numpy=True,
    )
    client.upsert(
        collection_name=COLLECTION,
        points=[
            qm.PointStruct(
                id=d.id,
                vector=vectors[i].tolist(),
                payload={"title": d.title, "content": d.content},
            )
            for i, d in enumerate(docs)
        ],
    )
    print(f"[boot] ingested {len(docs)} docs, dim={dim}")


print(f"[boot] device={DEVICE} model={MODEL_NAME}")
MODEL = SentenceTransformer(MODEL_NAME, device=DEVICE)
CLIENT = QdrantClient(url=QDRANT_URL)
wait_for_qdrant(CLIENT)
DOCS = load_docs()
DOC_BY_ID = {d.id: d for d in DOCS}
TOKENIZED = [tokenize(f"{d.title} {d.content}") for d in DOCS]
BM25 = BM25Okapi(TOKENIZED)
ingest(MODEL, CLIENT, DOCS)


def bm25_search(q: str, k: int) -> list[tuple[int, float]]:
    tokens = tokenize(q)
    if not tokens:
        return []
    scores = BM25.get_scores(tokens)
    ranked = sorted(
        ((DOCS[i].id, float(s)) for i, s in enumerate(scores) if s > 0),
        key=lambda x: x[1],
        reverse=True,
    )
    return ranked[:k]


def vector_search(q: str, k: int) -> list[tuple[int, float]]:
    qvec = MODEL.encode(
        f"query: {q}",
        normalize_embeddings=True,
        convert_to_numpy=True,
    )
    result = CLIENT.query_points(
        collection_name=COLLECTION,
        query=qvec.tolist(),
        limit=k,
        with_payload=False,
    )
    return [(int(p.id), float(p.score)) for p in result.points]


def rrf_fuse(
    bm25: list[tuple[int, float]],
    vec: list[tuple[int, float]],
    k: int,
) -> list[tuple[int, float, int | None, int | None]]:
    """Reciprocal Rank Fusion: score = sum(1 / (RRF_K + rank_i))."""
    bm25_rank = {doc_id: i + 1 for i, (doc_id, _) in enumerate(bm25)}
    vec_rank = {doc_id: i + 1 for i, (doc_id, _) in enumerate(vec)}
    all_ids = set(bm25_rank) | set(vec_rank)

    scored = []
    for doc_id in all_ids:
        score = 0.0
        if doc_id in bm25_rank:
            score += 1.0 / (RRF_K + bm25_rank[doc_id])
        if doc_id in vec_rank:
            score += 1.0 / (RRF_K + vec_rank[doc_id])
        scored.append((doc_id, score, bm25_rank.get(doc_id), vec_rank.get(doc_id)))
    scored.sort(key=lambda x: x[1], reverse=True)
    return scored[:k]


app = FastAPI(title="Hybrid Search (BM25 + vector via RRF)")
app.add_middleware(CORSMiddleware, allow_origins=["*"], allow_methods=["*"], allow_headers=["*"])


@app.get("/health")
def health() -> dict[str, str]:
    info = CLIENT.get_collection(COLLECTION)
    return {"status": "ok", "device": DEVICE, "model": MODEL_NAME, "points": str(info.points_count)}


@app.get("/search", response_model=SearchResponse)
def search(
    q: str = Query(..., min_length=1),
    k: int = Query(5, ge=1, le=20),
    mode: str = Query("hybrid", pattern="^(hybrid|bm25|vector)$"),
    pool: int = Query(20, ge=1, le=50),
) -> SearchResponse:
    t0 = time.perf_counter()

    if mode == "bm25":
        ranked = bm25_search(q, k)
        hits = [
            SearchHit(id=i, title=DOC_BY_ID[i].title, content=DOC_BY_ID[i].content, score=s, bm25_rank=idx + 1)
            for idx, (i, s) in enumerate(ranked)
        ]
    elif mode == "vector":
        ranked = vector_search(q, k)
        hits = [
            SearchHit(id=i, title=DOC_BY_ID[i].title, content=DOC_BY_ID[i].content, score=s, vector_rank=idx + 1)
            for idx, (i, s) in enumerate(ranked)
        ]
    else:
        bm = bm25_search(q, pool)
        vc = vector_search(q, pool)
        fused = rrf_fuse(bm, vc, k)
        hits = [
            SearchHit(
                id=i,
                title=DOC_BY_ID[i].title,
                content=DOC_BY_ID[i].content,
                score=round(s, 6),
                bm25_rank=br,
                vector_rank=vr,
            )
            for i, s, br, vr in fused
        ]

    ms = (time.perf_counter() - t0) * 1000
    return SearchResponse(query=q, hits=hits, ms=round(ms, 1), mode=mode)


if FRONTEND_DIR.is_dir():
    app.mount("/", StaticFiles(directory=str(FRONTEND_DIR), html=True), name="frontend")
