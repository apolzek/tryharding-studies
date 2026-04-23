from app.channels import pick_channels


def test_customer_event_goes_to_email():
    chs = pick_channels("customer.created")
    assert [c.name for c in chs] == ["email"]


def test_payment_failed_notifies_email_and_sms():
    assert {c.name for c in pick_channels("payment.failed")} == {"email", "sms"}


def test_payment_confirmed_notifies_email_and_push():
    assert {c.name for c in pick_channels("payment.confirmed")} == {"email", "push"}


def test_order_events_notify_email_and_push():
    assert {c.name for c in pick_channels("order.paid")} == {"email", "push"}
