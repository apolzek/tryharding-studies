"""Multi-topic consumer with DLQ + retry.

Binds one queue per upstream exchange (`customer.events`, `order.events`,
`payment.events`). Failed deliveries are retried up to `MAX_RETRIES`
via the message's own header; after that they're parked on a dead
letter exchange for manual inspection.
"""
from __future__ import annotations

import asyncio
import json
import logging
import os

import aio_pika
from aio_pika.abc import AbstractIncomingMessage

from .channels import Notification, pick_channels

log = logging.getLogger(__name__)

MAX_RETRIES = 3
SOURCES = {
    # exchange → (queue, binding-pattern)
    "customer.events": ("notif.customer", "customer.#"),
    "order.events":    ("notif.order",    "order.#"),
    "payment.events":  ("notif.payment",  "payment.#"),
}


async def _setup(channel: aio_pika.Channel):
    # Dead letter infra: one shared DLX/DLQ.
    dlx = await channel.declare_exchange("notification.dlx", aio_pika.ExchangeType.FANOUT, durable=True)
    dlq = await channel.declare_queue("notification.dlq", durable=True)
    await dlq.bind(dlx)

    bindings = []
    for source_exchange, (qname, rk) in SOURCES.items():
        ex = await channel.declare_exchange(source_exchange, aio_pika.ExchangeType.TOPIC, durable=True)
        q = await channel.declare_queue(qname, durable=True)
        await q.bind(ex, routing_key=rk)
        bindings.append((q, source_exchange))
    return bindings, dlx


async def run_consumer(redis_client) -> None:
    url = os.getenv("RABBIT_URL", "amqp://guest:guest@rabbitmq:5672/")
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
    await channel.set_qos(prefetch_count=32)
    bindings, dlx = await _setup(channel)

    log.info("notification consumer bound: %s", [q.name for q, _ in bindings])

    async def handle(q, source):
        async with q.iterator() as it:
            async for message in it:  # type: AbstractIncomingMessage
                try:
                    payload = json.loads(message.body)
                    event_type = message.routing_key
                    await _deliver(redis_client, event_type, payload)
                    await message.ack()
                except Exception as exc:
                    attempts = int((message.headers or {}).get("x-retry", 0)) + 1
                    if attempts >= MAX_RETRIES:
                        log.error("parking on DLQ after %s attempts: %s", attempts, exc)
                        await dlx.publish(
                            aio_pika.Message(
                                body=message.body,
                                headers={
                                    "x-original-rk": message.routing_key,
                                    "x-original-exchange": source,
                                    "x-error": str(exc),
                                },
                            ),
                            routing_key="",
                        )
                        await message.ack()
                    else:
                        log.warning("retry %s for %s: %s", attempts, message.routing_key, exc)
                        await message.nack(requeue=False)
                        await channel.default_exchange.publish(
                            aio_pika.Message(
                                body=message.body,
                                headers={"x-retry": attempts},
                            ),
                            routing_key=q.name,
                        )

    await asyncio.gather(*[handle(q, s) for q, s in bindings])


async def _deliver(redis_client, event_type: str, payload: dict) -> None:
    target = payload.get("email") or payload.get("customer_id") or payload.get("order_id") or "unknown"
    subject = f"[{event_type}]"
    body = json.dumps(payload, default=str)
    n = Notification(event_type=event_type, channel="multi", target=target, subject=subject, body=body)
    for ch in pick_channels(event_type):
        await ch.send(redis_client, Notification(event_type, ch.name, n.target, n.subject, n.body))
