"""Outbox worker — drains unsent outbox rows to RabbitMQ.

Guarantees at-least-once delivery of domain events while keeping
the atomicity of state change + event emission (both happen in the
same Postgres transaction).
"""
from __future__ import annotations

import asyncio
import json
import logging
import os

import aio_pika
import asyncpg

log = logging.getLogger(__name__)


async def run_outbox(pool: asyncpg.Pool) -> None:
    url = os.getenv("RABBIT_URL", "amqp://guest:guest@rabbitmq:5672/")
    for attempt in range(30):
        try:
            connection = await aio_pika.connect_robust(url)
            break
        except Exception as exc:
            log.warning("rabbit not ready (%s); retry %s", exc, attempt)
            await asyncio.sleep(2)
    else:
        log.error("outbox: giving up on rabbit")
        return

    channel = await connection.channel()
    declared: dict[str, aio_pika.Exchange] = {}

    async def exchange_for(name: str) -> aio_pika.Exchange:
        if name not in declared:
            declared[name] = await channel.declare_exchange(
                name, aio_pika.ExchangeType.TOPIC, durable=True
            )
        return declared[name]

    log.info("outbox worker started")

    while True:
        try:
            async with pool.acquire() as conn:
                rows = await conn.fetch(
                    "SELECT id, topic, routing_key, payload FROM outbox "
                    "WHERE sent_at IS NULL ORDER BY id LIMIT 50 FOR UPDATE SKIP LOCKED"
                )
                for r in rows:
                    ex = await exchange_for(r["topic"])
                    payload = r["payload"]
                    if isinstance(payload, str):
                        payload_bytes = payload.encode()
                    else:
                        payload_bytes = json.dumps(payload).encode()
                    await ex.publish(
                        aio_pika.Message(body=payload_bytes, content_type="application/json"),
                        routing_key=r["routing_key"],
                    )
                    await conn.execute(
                        "UPDATE outbox SET sent_at=now() WHERE id=$1", r["id"]
                    )
        except Exception:
            log.exception("outbox iteration failed")

        await asyncio.sleep(1)
