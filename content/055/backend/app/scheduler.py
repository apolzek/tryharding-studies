from __future__ import annotations

import logging
from datetime import datetime

from apscheduler.schedulers.background import BackgroundScheduler
from sqlalchemy.exc import IntegrityError

from .config import SCRAPE_INTERVAL_MINUTES
from .db import SessionLocal
from .models import Event, ScrapeRun, Target
from .scrapers import scrape_target

log = logging.getLogger("spy.scheduler")


def run_once(target_id: int | None = None) -> dict:
    """Scrape all networks for one target (or every target if id is None).
    Returns a summary dict with per-target counts."""
    summary: dict[str, dict[str, int]] = {}
    with SessionLocal() as db:
        q = db.query(Target)
        if target_id is not None:
            q = q.filter(Target.id == target_id)
        targets = q.all()
        for target in targets:
            summary[target.handle] = {}
            for network in target.networks or []:
                count = _scrape_one(db, target, network)
                summary[target.handle][network] = count
            target.last_scraped_at = datetime.utcnow()
            db.add(target)
        db.commit()
    return summary


def _scrape_one(db, target: Target, network: str) -> int:
    try:
        events = scrape_target(network, target.handle)
    except Exception as e:
        log.exception("scrape failed %s/%s: %s", target.handle, network, e)
        db.add(
            ScrapeRun(
                target_id=target.id,
                network=network,
                success=False,
                message=str(e)[:500],
                new_events=0,
            )
        )
        return 0
    new_count = 0
    for evt in events:
        row = Event(
            target_id=target.id,
            network=evt.network,
            kind=evt.kind,
            external_id=evt.external_id,
            title=evt.title,
            url=evt.url,
            body=evt.body,
            happened_at=evt.happened_at,
        )
        db.add(row)
        try:
            db.flush()
            new_count += 1
        except IntegrityError:
            db.rollback()
    db.add(
        ScrapeRun(
            target_id=target.id,
            network=network,
            success=True,
            new_events=new_count,
        )
    )
    return new_count


_scheduler: BackgroundScheduler | None = None


def start_scheduler() -> BackgroundScheduler:
    global _scheduler
    if _scheduler is not None:
        return _scheduler
    sched = BackgroundScheduler(timezone="UTC")
    sched.add_job(
        run_once,
        "interval",
        minutes=SCRAPE_INTERVAL_MINUTES,
        id="scrape-all",
        next_run_time=datetime.utcnow(),
        max_instances=1,
        coalesce=True,
    )
    sched.start()
    _scheduler = sched
    log.info("scheduler started, interval=%smin", SCRAPE_INTERVAL_MINUTES)
    return sched


def stop_scheduler() -> None:
    global _scheduler
    if _scheduler is not None:
        _scheduler.shutdown(wait=False)
        _scheduler = None
