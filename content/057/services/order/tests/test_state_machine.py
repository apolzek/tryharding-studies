import pytest

from app.state_machine import assert_transition, InvalidTransition


def test_happy_path():
    assert_transition("CREATED", "AWAITING_PAYMENT")
    assert_transition("AWAITING_PAYMENT", "PAID")
    assert_transition("PAID", "FULFILLED")
    assert_transition("FULFILLED", "DELIVERED")


def test_payment_retry_path():
    assert_transition("AWAITING_PAYMENT", "PAYMENT_FAILED")
    assert_transition("PAYMENT_FAILED", "AWAITING_PAYMENT")


def test_rejects_skipping_states():
    with pytest.raises(InvalidTransition):
        assert_transition("CREATED", "PAID")


def test_delivered_is_terminal():
    with pytest.raises(InvalidTransition):
        assert_transition("DELIVERED", "CANCELLED")


def test_cancelled_is_terminal():
    with pytest.raises(InvalidTransition):
        assert_transition("CANCELLED", "PAID")
