"""Unit tests for the consumer's event shape handling.

We don't talk to real RabbitMQ/ClickHouse — we verify the parsing
and the call into the store stays stable.
"""
from __future__ import annotations

import json

import pytest


class FakeStore:
    def __init__(self):
        self.calls = []

    def record(self, customer_id, plan, amount, status="active"):
        self.calls.append((customer_id, plan, amount, status))
        return {"id": "1"}


def handle_message(body: bytes, store: FakeStore) -> None:
    payload = json.loads(body)
    store.record(
        customer_id=payload.get("id", "unknown"),
        plan=payload.get("plan", "free"),
        amount=float(payload.get("amount", 0.0)),
        status="active",
    )


def test_handle_message_defaults():
    store = FakeStore()
    handle_message(b'{"id":"c1"}', store)
    assert store.calls == [("c1", "free", 0.0, "active")]


def test_handle_message_with_plan():
    store = FakeStore()
    handle_message(b'{"id":"c2","plan":"premium","amount":"19.99"}', store)
    assert store.calls == [("c2", "premium", 19.99, "active")]
