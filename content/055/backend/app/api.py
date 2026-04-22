from __future__ import annotations

from datetime import datetime, timedelta
from typing import Optional

from fastapi import APIRouter, Depends, HTTPException
from pydantic import BaseModel, Field, field_validator
from sqlalchemy.orm import Session

from .config import SUPPORTED_NETWORKS
from .db import get_session
from .digest import build_digest
from .models import Event, ScrapeRun, Target
from .scheduler import run_once

router = APIRouter()


class TargetIn(BaseModel):
    handle: str = Field(..., min_length=1, max_length=128)
    networks: list[str] = Field(default_factory=list)

    @field_validator("handle")
    @classmethod
    def strip_at(cls, v: str) -> str:
        return v.strip().lstrip("@")

    @field_validator("networks")
    @classmethod
    def validate_networks(cls, v: list[str]) -> list[str]:
        bad = [n for n in v if n not in SUPPORTED_NETWORKS]
        if bad:
            raise ValueError(f"Unsupported networks: {bad}. Allowed: {SUPPORTED_NETWORKS}")
        return v


class TargetOut(BaseModel):
    id: int
    handle: str
    networks: list[str]
    created_at: datetime
    last_scraped_at: Optional[datetime]


class EventOut(BaseModel):
    id: int
    target_id: int
    network: str
    kind: str
    title: str
    url: str
    body: Optional[str]
    happened_at: datetime
    fetched_at: datetime


@router.get("/networks")
def list_networks():
    return {"supported": SUPPORTED_NETWORKS}


@router.post("/targets", response_model=TargetOut)
def create_target(body: TargetIn, db: Session = Depends(get_session)):
    existing = db.query(Target).filter(Target.handle == body.handle).first()
    if existing:
        networks = sorted(set(existing.networks or []) | set(body.networks))
        existing.networks = networks
        db.add(existing)
        db.commit()
        db.refresh(existing)
        return _target_to_out(existing)
    row = Target(handle=body.handle, networks=body.networks)
    db.add(row)
    db.commit()
    db.refresh(row)
    return _target_to_out(row)


@router.get("/targets", response_model=list[TargetOut])
def list_targets(db: Session = Depends(get_session)):
    rows = db.query(Target).order_by(Target.created_at.desc()).all()
    return [_target_to_out(r) for r in rows]


@router.delete("/targets/{target_id}")
def delete_target(target_id: int, db: Session = Depends(get_session)):
    row = db.get(Target, target_id)
    if not row:
        raise HTTPException(404, "target not found")
    db.delete(row)
    db.commit()
    return {"deleted": target_id}


@router.post("/targets/{target_id}/scrape")
def scrape_now(target_id: int, db: Session = Depends(get_session)):
    row = db.get(Target, target_id)
    if not row:
        raise HTTPException(404, "target not found")
    summary = run_once(target_id=target_id)
    return {"target": row.handle, "summary": summary.get(row.handle, {})}


@router.get("/targets/{target_id}/events", response_model=list[EventOut])
def list_events(
    target_id: int,
    limit: int = 100,
    since_hours: Optional[int] = None,
    network: Optional[str] = None,
    db: Session = Depends(get_session),
):
    q = db.query(Event).filter(Event.target_id == target_id)
    if since_hours is not None:
        q = q.filter(Event.happened_at >= datetime.utcnow() - timedelta(hours=since_hours))
    if network:
        q = q.filter(Event.network == network)
    rows = q.order_by(Event.happened_at.desc()).limit(limit).all()
    return [
        EventOut(
            id=r.id,
            target_id=r.target_id,
            network=r.network,
            kind=r.kind,
            title=r.title,
            url=r.url,
            body=r.body,
            happened_at=r.happened_at,
            fetched_at=r.fetched_at,
        )
        for r in rows
    ]


@router.get("/targets/{target_id}/digest")
def get_digest(target_id: int, hours: int = 24, db: Session = Depends(get_session)):
    row = db.get(Target, target_id)
    if not row:
        raise HTTPException(404, "target not found")
    return build_digest(db, row, hours=hours)


@router.get("/targets/{target_id}/runs")
def list_runs(target_id: int, limit: int = 30, db: Session = Depends(get_session)):
    rows = (
        db.query(ScrapeRun)
        .filter(ScrapeRun.target_id == target_id)
        .order_by(ScrapeRun.run_at.desc())
        .limit(limit)
        .all()
    )
    return [
        {
            "id": r.id,
            "network": r.network,
            "run_at": r.run_at.isoformat(),
            "success": r.success,
            "message": r.message,
            "new_events": r.new_events,
        }
        for r in rows
    ]


def _target_to_out(row: Target) -> TargetOut:
    return TargetOut(
        id=row.id,
        handle=row.handle,
        networks=row.networks or [],
        created_at=row.created_at,
        last_scraped_at=row.last_scraped_at,
    )
