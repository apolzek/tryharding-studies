"""RabbitMQ consumer for customer.created events.

Implements the write-side of the CQRS split: materializes a
signature record into the columnar store whenever a new customer
is created.
"""
from __future__ import annotations

import asyncio
import json
import logging
import os

import aio_pika

from .store import SignatureStore

log = logging.getLogger(__name__)


async def run_consumer(store: SignatureStore) -> None:
    url = os.getenv("RABBIT_URL", "amqp://guest:guest@rabbitmq:5672/")
    exchange = os.getenv("RABBIT_EXCHANGE", "customer.events")
    queue_name = os.getenv("RABBIT_QUEUE", "signature.customer-events")

    for attempt in range(30):
        try:
            connection = await aio_pika.connect_robust(url)
            break
        except Exception as exc:
            log.warning("rabbit not ready (%s); retry %s", exc, attempt)
            await asyncio.sleep(2)
    else:
        log.error("giving up on rabbit")
        return

    channel = await connection.channel()
    ex = await channel.declare_exchange(exchange, aio_pika.ExchangeType.TOPIC, durable=True)
    q = await channel.declare_queue(queue_name, durable=True)
    await q.bind(ex, routing_key="customer.#")

    log.info("signature consumer bound to %s", exchange)

    async with q.iterator() as it:
        async for message in it:
            async with message.process():
                try:
                    payload = json.loads(message.body)
                    store.record(
                        customer_id=payload.get("id", "unknown"),
                        plan=payload.get("plan", "free"),
                        amount=float(payload.get("amount", 0.0)),
                        status="active",
                    )
                except Exception:
                    log.exception("failed to process event")
