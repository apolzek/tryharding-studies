"""BM25 keyword search over a small corpus of tech documents (PT-BR)."""
from __future__ import annotations

import json
import re
from pathlib import Path

from fastapi import FastAPI, Query
from fastapi.middleware.cors import CORSMiddleware
from fastapi.staticfiles import StaticFiles
from pydantic import BaseModel
from rank_bm25 import BM25Okapi
from unidecode import unidecode

DOCS_PATH = Path(__file__).parent / "docs.json"
FRONTEND_DIR = Path(__file__).parent / "frontend"

PT_STOPWORDS = {
    "a", "o", "as", "os", "um", "uma", "uns", "umas", "de", "do", "da", "dos", "das",
    "em", "no", "na", "nos", "nas", "por", "para", "com", "sem", "que", "e", "ou",
    "mas", "se", "como", "ao", "aos", "à", "às", "é", "foi", "ser", "são", "este",
    "esta", "isso", "isto", "seu", "sua", "seus", "suas", "sobre", "entre", "também",
}

TOKEN_RE = re.compile(r"[a-z0-9]+")


def tokenize(text: str) -> list[str]:
    """Lowercase, strip accents, split on non-alphanumeric, drop stopwords."""
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


class SearchResponse(BaseModel):
    query: str
    hits: list[SearchHit]


def load_docs() -> list[Doc]:
    with DOCS_PATH.open(encoding="utf-8") as f:
        return [Doc(**d) for d in json.load(f)]


DOCS = load_docs()
TOKENIZED = [tokenize(f"{d.title} {d.content}") for d in DOCS]
BM25 = BM25Okapi(TOKENIZED)

app = FastAPI(title="Keyword Search (BM25)")
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["*"],
    allow_headers=["*"],
)


@app.get("/health")
def health() -> dict[str, str]:
    return {"status": "ok", "docs": str(len(DOCS))}


@app.get("/search", response_model=SearchResponse)
def search(q: str = Query(..., min_length=1), k: int = Query(5, ge=1, le=20)) -> SearchResponse:
    tokens = tokenize(q)
    if not tokens:
        return SearchResponse(query=q, hits=[])

    scores = BM25.get_scores(tokens)
    ranked = sorted(enumerate(scores), key=lambda x: x[1], reverse=True)
    hits = [
        SearchHit(
            id=DOCS[i].id,
            title=DOCS[i].title,
            content=DOCS[i].content,
            score=float(score),
        )
        for i, score in ranked[:k]
        if score > 0
    ]
    return SearchResponse(query=q, hits=hits)


@app.get("/docs-list", response_model=list[Doc])
def list_docs() -> list[Doc]:
    return DOCS


if FRONTEND_DIR.is_dir():
    app.mount("/", StaticFiles(directory=str(FRONTEND_DIR), html=True), name="frontend")
