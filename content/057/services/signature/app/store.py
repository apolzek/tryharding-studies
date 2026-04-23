"""ClickHouse-backed append-only store for signature events.

Columnar storage is a good fit here: we only read via aggregations
(counts per plan, revenue per day, retention) and write heavy
volumes of immutable events.
"""
from __future__ import annotations

import datetime as dt
import uuid
from typing import Any

import clickhouse_connect


DDL = """
CREATE TABLE IF NOT EXISTS signatures (
    id UUID,
    customer_id String,
    plan String,
    amount Float64,
    status Enum8('pending'=1, 'active'=2, 'cancelled'=3),
    created_at DateTime64(3)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(created_at)
ORDER BY (plan, created_at, id)
"""


class SignatureStore:
    def __init__(self, host: str, port: int, db: str):
        self.client = clickhouse_connect.get_client(host=host, port=port, database=db)
        self.client.command(f"CREATE DATABASE IF NOT EXISTS {db}")
        self.client.command(DDL)

    def record(self, customer_id: str, plan: str, amount: float, status: str = "active") -> dict[str, Any]:
        sig_id = uuid.uuid4()
        created = dt.datetime.utcnow()
        self.client.insert(
            "signatures",
            [[sig_id, customer_id, plan, float(amount), status, created]],
            column_names=["id", "customer_id", "plan", "amount", "status", "created_at"],
        )
        return {
            "id": str(sig_id),
            "customer_id": customer_id,
            "plan": plan,
            "amount": amount,
            "status": status,
            "created_at": created.isoformat(),
        }

    def by_customer(self, customer_id: str) -> list[dict[str, Any]]:
        rows = self.client.query(
            "SELECT id, customer_id, plan, amount, status, created_at "
            "FROM signatures WHERE customer_id=%(c)s ORDER BY created_at DESC",
            parameters={"c": customer_id},
        ).result_rows
        return [self._row_to_dict(r) for r in rows]

    def summary(self) -> list[dict[str, Any]]:
        rows = self.client.query(
            "SELECT plan, count() AS total, sum(amount) AS revenue "
            "FROM signatures WHERE status='active' GROUP BY plan ORDER BY total DESC"
        ).result_rows
        return [{"plan": p, "total": int(t), "revenue": float(r)} for (p, t, r) in rows]

    @staticmethod
    def _row_to_dict(r) -> dict[str, Any]:
        return {
            "id": str(r[0]),
            "customer_id": r[1],
            "plan": r[2],
            "amount": float(r[3]),
            "status": r[4],
            "created_at": r[5].isoformat() if hasattr(r[5], "isoformat") else r[5],
        }
