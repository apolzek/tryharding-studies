"""Notification service — consumes and fans out to channels."""
from __future__ import annotations

import asyncio
import json
import os
from contextlib import asynccontextmanager

import redis.asyncio as aioredis
from fastapi import FastAPI
from opentelemetry.instrumentation.fastapi import FastAPIInstrumentor

from .consumer import run_consumer


REDIS_URL = os.getenv("REDIS_URL", "redis://redis:6379/0")


@asynccontextmanager
async def lifespan(app: FastAPI):
    r = aioredis.from_url(REDIS_URL, decode_responses=True)
    app.state.redis = r
    task = asyncio.create_task(run_consumer(r))
    try:
        yield
    finally:
        task.cancel()
        await r.close()


app = FastAPI(title="notification-service", lifespan=lifespan)
FastAPIInstrumentor.instrument_app(app)


@app.get("/health")
async def health():
    return {"status": "ok"}


@app.get("/notifications/recent")
async def recent(limit: int = 50):
    """Tail of recent notifications (read from Redis list)."""
    items = await app.state.redis.lrange("notifications:stream", 0, limit - 1)
    return [json.loads(i) for i in items]
