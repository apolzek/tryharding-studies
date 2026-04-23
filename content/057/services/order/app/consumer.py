"""Consumes payment events and drives the order state machine."""
from __future__ import annotations

import asyncio
import json
import logging
import os

import aio_pika

from .repository import OrderRepository
from .state_machine import InvalidTransition

log = logging.getLogger(__name__)


async def run_payment_consumer(repo: OrderRepository) -> None:
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
    ex = await channel.declare_exchange("payment.events", aio_pika.ExchangeType.TOPIC, durable=True)
    q = await channel.declare_queue("order.payment-events", durable=True)
    await q.bind(ex, routing_key="payment.#")

    log.info("order ← payment.events consumer ready")

    async with q.iterator() as it:
        async for message in it:
            async with message.process():
                try:
                    payload = json.loads(message.body)
                    order_id = payload.get("order_id")
                    if not order_id:
                        continue
                    if message.routing_key == "payment.confirmed":
                        await repo.transition(
                            order_id, "PAID",
                            event_payload={"transaction_id": payload.get("transaction_id")},
                            emit_topic=("order.events", "order.paid"),
                        )
                    elif message.routing_key == "payment.failed":
                        await repo.transition(
                            order_id, "PAYMENT_FAILED",
                            event_payload={"reason": payload.get("reason", "unknown")},
                            emit_topic=("order.events", "order.payment_failed"),
                        )
                except InvalidTransition as e:
                    log.warning("ignored invalid transition: %s", e)
                except Exception:
                    log.exception("failed to handle payment event")
