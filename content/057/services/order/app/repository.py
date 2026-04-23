"""Order repository — current-state + event-sourced append.

Every mutation is done inside a transaction that also writes an
entry to `order_events` (history) and, when relevant, an `outbox`
row. This gives us atomic "change state + emit domain event".
"""
from __future__ import annotations

import json
import uuid
from typing import Any

import asyncpg

from .state_machine import assert_transition


class OrderRepository:
    def __init__(self, pool: asyncpg.Pool):
        self.pool = pool

    async def create(
        self,
        customer_id: str,
        items: list[dict[str, Any]],
    ) -> dict[str, Any]:
        order_id = uuid.uuid4()
        total = sum(float(i["qty"]) * float(i["unit_price"]) for i in items)

        async with self.pool.acquire() as conn:
            async with conn.transaction():
                await conn.execute(
                    "INSERT INTO orders (id, customer_id, status, total) VALUES ($1,$2,'CREATED',$3)",
                    order_id, customer_id, total,
                )
                for it in items:
                    await conn.execute(
                        "INSERT INTO order_items (order_id, product_id, sku, qty, unit_price) "
                        "VALUES ($1,$2,$3,$4,$5)",
                        order_id, it["product_id"], it["sku"], int(it["qty"]), float(it["unit_price"]),
                    )
                await self._append_event(
                    conn, order_id, "OrderCreated",
                    {"customer_id": customer_id, "total": total, "items": items},
                )
                await self._enqueue_outbox(
                    conn, "order.events", "order.created",
                    {"order_id": str(order_id), "customer_id": customer_id, "total": total},
                )

        return await self.get(order_id)

    async def get(self, order_id: uuid.UUID | str) -> dict[str, Any] | None:
        if isinstance(order_id, str):
            order_id = uuid.UUID(order_id)
        async with self.pool.acquire() as conn:
            row = await conn.fetchrow("SELECT * FROM orders WHERE id=$1", order_id)
            if not row:
                return None
            items = await conn.fetch("SELECT * FROM order_items WHERE order_id=$1", order_id)
        d = dict(row)
        d["id"] = str(d["id"])
        d["items"] = [dict(i) for i in items]
        for i in d["items"]:
            i["order_id"] = str(i["order_id"])
        return d

    async def list_by_customer(self, customer_id: str) -> list[dict[str, Any]]:
        async with self.pool.acquire() as conn:
            rows = await conn.fetch(
                "SELECT * FROM orders WHERE customer_id=$1 ORDER BY created_at DESC LIMIT 100",
                customer_id,
            )
        out = []
        for r in rows:
            d = dict(r)
            d["id"] = str(d["id"])
            out.append(d)
        return out

    async def transition(
        self,
        order_id: uuid.UUID | str,
        to_state: str,
        event_payload: dict[str, Any] | None = None,
        emit_topic: tuple[str, str] | None = None,
    ) -> dict[str, Any]:
        if isinstance(order_id, str):
            order_id = uuid.UUID(order_id)

        async with self.pool.acquire() as conn:
            async with conn.transaction():
                current = await conn.fetchrow(
                    "SELECT status, version FROM orders WHERE id=$1 FOR UPDATE",
                    order_id,
                )
                if not current:
                    raise KeyError("order not found")
                assert_transition(current["status"], to_state)
                await conn.execute(
                    "UPDATE orders SET status=$1, version=version+1, updated_at=now() WHERE id=$2",
                    to_state, order_id,
                )
                await self._append_event(
                    conn, order_id,
                    f"OrderTransitioned:{to_state}",
                    {"from": current["status"], "to": to_state, **(event_payload or {})},
                )
                if emit_topic:
                    topic, rk = emit_topic
                    await self._enqueue_outbox(
                        conn, topic, rk,
                        {"order_id": str(order_id), "status": to_state, **(event_payload or {})},
                    )
        return await self.get(order_id)

    async def history(self, order_id: uuid.UUID | str) -> list[dict[str, Any]]:
        """Event sourcing: replayable log of everything that ever happened."""
        if isinstance(order_id, str):
            order_id = uuid.UUID(order_id)
        async with self.pool.acquire() as conn:
            rows = await conn.fetch(
                "SELECT id, event_type, payload, occurred_at FROM order_events "
                "WHERE order_id=$1 ORDER BY id",
                order_id,
            )
        return [
            {
                "seq": r["id"],
                "event_type": r["event_type"],
                "payload": json.loads(r["payload"]) if isinstance(r["payload"], str) else r["payload"],
                "occurred_at": r["occurred_at"].isoformat(),
            }
            for r in rows
        ]

    @staticmethod
    async def _append_event(conn, order_id, event_type: str, payload: dict[str, Any]) -> None:
        await conn.execute(
            "INSERT INTO order_events (order_id, event_type, payload) VALUES ($1,$2,$3::jsonb)",
            order_id, event_type, json.dumps(payload),
        )

    @staticmethod
    async def _enqueue_outbox(conn, topic: str, routing_key: str, payload: dict[str, Any]) -> None:
        await conn.execute(
            "INSERT INTO outbox (topic, routing_key, payload) VALUES ($1,$2,$3::jsonb)",
            topic, routing_key, json.dumps(payload),
        )
