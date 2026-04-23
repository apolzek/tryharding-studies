"""Catalog service — FastAPI + MongoDB.

Exposes CRUD on catalogs/categories. Non-relational storage fits the
flexible shape of catalog metadata (nested categories, tags, images).
"""
from __future__ import annotations

import os
from contextlib import asynccontextmanager
from typing import Any

from fastapi import FastAPI, HTTPException
from motor.motor_asyncio import AsyncIOMotorClient
from opentelemetry.instrumentation.fastapi import FastAPIInstrumentor
from opentelemetry.instrumentation.pymongo import PymongoInstrumentor
from pydantic import BaseModel, Field

from .repository import CatalogRepository


class CatalogIn(BaseModel):
    name: str = Field(min_length=1)
    description: str | None = None
    categories: list[str] = []
    metadata: dict[str, Any] = {}


class Catalog(CatalogIn):
    id: str


MONGO_URL = os.getenv("MONGO_URL", "mongodb://mongo:27017")
DB_NAME = os.getenv("MONGO_DB", "catalog")


@asynccontextmanager
async def lifespan(app: FastAPI):
    client = AsyncIOMotorClient(MONGO_URL)
    db = client[DB_NAME]
    app.state.repo = CatalogRepository(db)
    yield
    client.close()


app = FastAPI(title="catalog-service", lifespan=lifespan)
FastAPIInstrumentor.instrument_app(app)
PymongoInstrumentor().instrument()


@app.get("/health")
async def health():
    return {"status": "ok"}


@app.post("/catalogs", response_model=Catalog, status_code=201)
async def create_catalog(payload: CatalogIn):
    repo: CatalogRepository = app.state.repo
    return await repo.create(payload.model_dump())


@app.get("/catalogs", response_model=list[Catalog])
async def list_catalogs():
    repo: CatalogRepository = app.state.repo
    return await repo.list()


@app.get("/catalogs/{catalog_id}", response_model=Catalog)
async def get_catalog(catalog_id: str):
    repo: CatalogRepository = app.state.repo
    doc = await repo.get(catalog_id)
    if not doc:
        raise HTTPException(404, "catalog not found")
    return doc


@app.delete("/catalogs/{catalog_id}", status_code=204)
async def delete_catalog(catalog_id: str):
    repo: CatalogRepository = app.state.repo
    await repo.delete(catalog_id)
    return None
