"""Delivery channels — each one is a Strategy picked per event type.

In a real system these would be SendGrid / Twilio / APNs / etc.
Here they log to stdout and push into a Redis list for the UI's
live-feed (SSE) to read. Whether a channel *fails* is deterministic
via a class-level flag, which the tests use.
"""
from __future__ import annotations

import json
import logging
import random
from dataclasses import dataclass

log = logging.getLogger(__name__)


@dataclass
class Notification:
    event_type: str
    channel: str
    target: str
    subject: str
    body: str


class Channel:
    name: str = "base"
    failure_rate: float = 0.0

    async def send(self, redis, n: Notification) -> None:
        if self.failure_rate and random.random() < self.failure_rate:
            raise RuntimeError(f"{self.name} transient failure")
        log.info("[%s → %s] %s :: %s", self.name, n.target, n.subject, n.body)
        if redis is not None:
            payload = {
                "channel": self.name,
                "target": n.target,
                "subject": n.subject,
                "body": n.body,
                "event": n.event_type,
            }
            await redis.lpush("notifications:stream", json.dumps(payload))
            await redis.ltrim("notifications:stream", 0, 199)


class EmailChannel(Channel):
    name = "email"


class SmsChannel(Channel):
    name = "sms"


class PushChannel(Channel):
    name = "push"


def pick_channels(event_type: str) -> list[Channel]:
    """Route events → channels."""
    if event_type.startswith("customer."):
        return [EmailChannel()]
    if event_type == "payment.confirmed":
        return [EmailChannel(), PushChannel()]
    if event_type == "payment.failed":
        return [EmailChannel(), SmsChannel()]
    if event_type.startswith("order."):
        return [EmailChannel(), PushChannel()]
    return [EmailChannel()]
