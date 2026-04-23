"""Order state machine.

Validates legal transitions only. Failing a transition raises
`InvalidTransition` — the caller should treat it as a 409 Conflict.
"""
from __future__ import annotations


class InvalidTransition(Exception):
    pass


TRANSITIONS: dict[str, set[str]] = {
    "CREATED": {"AWAITING_PAYMENT", "CANCELLED"},
    "AWAITING_PAYMENT": {"PAID", "PAYMENT_FAILED", "CANCELLED"},
    "PAYMENT_FAILED": {"AWAITING_PAYMENT", "CANCELLED"},
    "PAID": {"FULFILLED", "REFUNDED"},
    "FULFILLED": {"DELIVERED", "REFUNDED"},
    "DELIVERED": set(),
    "CANCELLED": set(),
    "REFUNDED": set(),
}


def assert_transition(from_state: str, to_state: str) -> None:
    allowed = TRANSITIONS.get(from_state, set())
    if to_state not in allowed:
        raise InvalidTransition(f"{from_state} -> {to_state} is not allowed")
