"""Dense vector search: multilingual-e5-small embeddings in Qdrant."""
from __future__ import annotations

import json
import os
import time
from pathlib import Path

import torch
from fastapi import FastAPI, Query
from fastapi.middleware.cors import CORSMiddleware
from fastapi.staticfiles import StaticFiles
from pydantic import BaseModel
from qdrant_client import QdrantClient
from qdrant_client.http import models as qm
from sentence_transformers import SentenceTransformer

DOCS_PATH = Path(__file__).parent / "docs.json"
FRONTEND_DIR = Path(__file__).parent / "frontend"

QDRANT_URL = os.environ.get("QDRANT_URL", "http://qdrant:6333")
COLLECTION = "docs"
MODEL_NAME = os.environ.get("EMBED_MODEL", "intfloat/multilingual-e5-small")
DEVICE = "cuda" if torch.cuda.is_available() else "cpu"

print(f"[boot] device={DEVICE} model={MODEL_NAME}")


class Doc(BaseModel):
    id: int
    title: str
    content: str


class SearchHit(BaseModel):
    id: int
    title: str
    content: str
    score: float


class SearchResponse(BaseModel):
    query: str
    hits: list[SearchHit]
    ms: float


def load_docs() -> list[Doc]:
    with DOCS_PATH.open(encoding="utf-8") as f:
        return [Doc(**d) for d in json.load(f)]


def wait_for_qdrant(client: QdrantClient, attempts: int = 30) -> None:
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


MODEL = SentenceTransformer(MODEL_NAME, device=DEVICE)
CLIENT = QdrantClient(url=QDRANT_URL)
wait_for_qdrant(CLIENT)
DOCS = load_docs()
ingest(MODEL, CLIENT, DOCS)

app = FastAPI(title="Vector Search (Qdrant + e5)")
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["*"],
    allow_headers=["*"],
)


@app.get("/health")
def health() -> dict[str, str]:
    info = CLIENT.get_collection(COLLECTION)
    return {
        "status": "ok",
        "device": DEVICE,
        "model": MODEL_NAME,
        "points": str(info.points_count),
    }


@app.get("/search", response_model=SearchResponse)
def search(q: str = Query(..., min_length=1), k: int = Query(5, ge=1, le=20)) -> SearchResponse:
    t0 = time.perf_counter()
    qvec = MODEL.encode(
        f"query: {q}",
        normalize_embeddings=True,
        convert_to_numpy=True,
    )
    result = CLIENT.query_points(
        collection_name=COLLECTION,
        query=qvec.tolist(),
        limit=k,
        with_payload=True,
    )
    hits = [
        SearchHit(
            id=int(p.id),
            title=p.payload["title"],
            content=p.payload["content"],
            score=float(p.score),
        )
        for p in result.points
    ]
    ms = (time.perf_counter() - t0) * 1000
    return SearchResponse(query=q, hits=hits, ms=round(ms, 1))


if FRONTEND_DIR.is_dir():
    app.mount("/", StaticFiles(directory=str(FRONTEND_DIR), html=True), name="frontend")
