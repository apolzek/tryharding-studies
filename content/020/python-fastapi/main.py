import os
import logging

import asyncpg
import httpx
from fastapi import FastAPI, Query, HTTPException
from pydantic import BaseModel

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# ── Config ────────────────────────────────────────────────────────────────────

RUST_AXUM_URL = os.getenv("RUST_AXUM_URL", "http://localhost:8083")
DATABASE_URL  = os.getenv("DATABASE_URL",  "postgresql://loandb:loandb@localhost:5432/loandb")

COMPLIANCE_LIMITS: dict[str, float] = {
    "USD": 200_000.0,
    "BRL": 200_000.0,
    "EUR": 200_000.0,
    "GBP": 200_000.0,
}

# ── Models ────────────────────────────────────────────────────────────────────

class EnrichedLoanRequest(BaseModel):
    application_id: str
    customer_id: str
    amount: float
    currency: str
    credit_score: int
    account_age_days: int
    monthly_income: float
    purpose: str | None = None


# ── App ───────────────────────────────────────────────────────────────────────
# All OTel instrumentation is injected automatically by the
# `opentelemetry-instrument` launcher (see Dockerfile CMD).
# No manual span creation or SDK bootstrap needed here.

app = FastAPI(title="python-fastapi – Currency & Compliance Service")


@app.post("/api/loan/compliance")
async def compliance_check(
    payload: EnrichedLoanRequest,
    currency: str = Query(default="USD"),
):
    # ── Compliance limit check ────────────────────────────────────────────────
    limit = COMPLIANCE_LIMITS.get(currency.upper(), 150_000.0)
    if payload.amount > limit:
        raise HTTPException(
            status_code=422,
            detail=f"Amount ${payload.amount:.2f} exceeds compliance limit ${limit:.2f}",
        )

    # ── Call rust-axum for fraud scoring ──────────────────────────────────────
    # httpx is auto-instrumented: traceparent header injected automatically
    score_payload = {
        "application_id":  payload.application_id,
        "customer_id":     payload.customer_id,
        "amount_usd":      round(payload.amount, 2),
        "credit_score":    payload.credit_score,
        "account_age_days": payload.account_age_days,
        "monthly_income":  payload.monthly_income,
        "currency":        currency,
    }
    async with httpx.AsyncClient(timeout=10.0) as client:
        rust_resp = await client.post(f"{RUST_AXUM_URL}/api/loan/score", json=score_payload)
        rust_resp.raise_for_status()
        score_result = rust_resp.json()

    # ── Persist decision to Postgres ──────────────────────────────────────────
    # asyncpg is auto-instrumented: DB span created automatically
    conn = await asyncpg.connect(DATABASE_URL)
    try:
        await conn.execute(
            """INSERT INTO loan_applications
                   (application_id, customer_id, amount, currency,
                    compliance_status, fraud_decision)
               VALUES ($1, $2, $3, $4, $5, $6)
               ON CONFLICT (application_id) DO NOTHING""",
            payload.application_id,
            payload.customer_id,
            payload.amount,
            currency,
            "PASSED",
            score_result.get("decision", "UNKNOWN"),
        )
    finally:
        await conn.close()

    logger.info(
        "application_id=%s compliance=PASSED fraud=%s",
        payload.application_id, score_result.get("decision"),
    )

    return {
        "application_id":       payload.application_id,
        "amount_usd":           round(payload.amount, 2),
        "compliance_limit_usd": limit,
        "compliance_status":    "PASSED",
        "fraud_assessment":     score_result,
    }


@app.get("/health")
async def health():
    return {"status": "ok", "service": "python-fastapi"}
