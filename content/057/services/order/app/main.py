"""Order service — orchestration + event sourcing + outbox."""
from __future__ import annotations

import asyncio
import os
from contextlib import asynccontextmanager

from fastapi import FastAPI, HTTPException
from opentelemetry.instrumentation.fastapi import FastAPIInstrumentor
from pydantic import BaseModel, Field

from .chaos import ChaosMiddleware, ChaosState
from .consumer import run_payment_consumer
from .db import make_pool
from .outbox import run_outbox
from .repository import OrderRepository
from .state_machine import InvalidTransition


class OrderItemIn(BaseModel):
    product_id: str
    sku: str
    qty: int = Field(gt=0)
    unit_price: float = Field(gt=0)


class OrderIn(BaseModel):
    customer_id: str
    items: list[OrderItemIn]


class ChaosIn(BaseModel):
    latency_ms: int = 0
    error_rate: float = 0.0


DSN = os.getenv(
    "DATABASE_URL",
    "postgres://postgres:postgres@postgres:5432/orderdb",
)

# Shared, long-lived chaos state — referenced by both the middleware
# (attached at app construction) and the HTTP handlers below.
chaos_state = ChaosState()


@asynccontextmanager
async def lifespan(app: FastAPI):
    pool = await make_pool(DSN)
    repo = OrderRepository(pool)
    app.state.repo = repo
    app.state.chaos = chaos_state
    tasks = [
        asyncio.create_task(run_outbox(pool)),
        asyncio.create_task(run_payment_consumer(repo)),
    ]
    try:
        yield
    finally:
        for t in tasks:
            t.cancel()
        await pool.close()


app = FastAPI(title="order-service", lifespan=lifespan)
app.add_middleware(ChaosMiddleware, state=chaos_state)
FastAPIInstrumentor.instrument_app(app)


@app.get("/health")
async def health():
    return {"status": "ok"}


@app.post("/orders", status_code=201)
async def create_order(payload: OrderIn):
    return await app.state.repo.create(
        payload.customer_id, [i.model_dump() for i in payload.items]
    )


@app.get("/orders/{order_id}")
async def get_order(order_id: str):
    o = await app.state.repo.get(order_id)
    if not o:
        raise HTTPException(404, "order not found")
    return o


@app.get("/orders/by-customer/{customer_id}")
async def list_by_customer(customer_id: str):
    return await app.state.repo.list_by_customer(customer_id)


@app.get("/orders/{order_id}/history")
async def order_history(order_id: str):
    """Event-sourced, replayable log — shows every state change."""
    return await app.state.repo.history(order_id)


@app.post("/orders/{order_id}/request-payment")
async def request_payment(order_id: str):
    try:
        return await app.state.repo.transition(
            order_id, "AWAITING_PAYMENT",
            emit_topic=("order.events", "order.payment_requested"),
        )
    except InvalidTransition as e:
        raise HTTPException(409, str(e))
    except KeyError:
        raise HTTPException(404, "order not found")


@app.post("/orders/{order_id}/fulfill")
async def fulfill(order_id: str):
    try:
        return await app.state.repo.transition(
            order_id, "FULFILLED",
            emit_topic=("order.events", "order.fulfilled"),
        )
    except InvalidTransition as e:
        raise HTTPException(409, str(e))
    except KeyError:
        raise HTTPException(404, "order not found")


@app.post("/orders/{order_id}/cancel")
async def cancel(order_id: str):
    try:
        return await app.state.repo.transition(
            order_id, "CANCELLED",
            emit_topic=("order.events", "order.cancelled"),
        )
    except InvalidTransition as e:
        raise HTTPException(409, str(e))
    except KeyError:
        raise HTTPException(404, "order not found")


@app.get("/chaos")
async def chaos_status():
    return app.state.chaos.to_dict()


@app.post("/chaos/inject")
async def chaos_inject(payload: ChaosIn):
    app.state.chaos.latency_ms = max(0, payload.latency_ms)
    app.state.chaos.error_rate = max(0.0, min(1.0, payload.error_rate))
    return app.state.chaos.to_dict()
