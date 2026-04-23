"""Signature service — CQRS read API + RabbitMQ write-side consumer.

- Reads (HTTP): `/signatures/summary`, `/signatures/by-customer/{id}`.
- Writes: background task consuming RabbitMQ customer events.
"""
from __future__ import annotations

import asyncio
import os
from contextlib import asynccontextmanager

from fastapi import FastAPI, HTTPException
from opentelemetry.instrumentation.fastapi import FastAPIInstrumentor
from pydantic import BaseModel

from .consumer import run_consumer
from .store import SignatureStore


class SignatureIn(BaseModel):
    customer_id: str
    plan: str
    amount: float = 0.0


@asynccontextmanager
async def lifespan(app: FastAPI):
    store = SignatureStore(
        host=os.getenv("CLICKHOUSE_HOST", "clickhouse"),
        port=int(os.getenv("CLICKHOUSE_PORT", "8123")),
        db=os.getenv("CLICKHOUSE_DB", "signatures"),
    )
    app.state.store = store
    task = asyncio.create_task(run_consumer(store))
    try:
        yield
    finally:
        task.cancel()


app = FastAPI(title="signature-service", lifespan=lifespan)
FastAPIInstrumentor.instrument_app(app)


@app.get("/health")
async def health():
    return {"status": "ok"}


@app.post("/signatures", status_code=201)
async def create_signature(payload: SignatureIn):
    return app.state.store.record(payload.customer_id, payload.plan, payload.amount)


@app.get("/signatures/summary")
async def summary():
    return app.state.store.summary()


@app.get("/signatures/by-customer/{customer_id}")
async def by_customer(customer_id: str):
    items = app.state.store.by_customer(customer_id)
    if not items:
        raise HTTPException(404, "no signatures for customer")
    return items
