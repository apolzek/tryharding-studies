from __future__ import annotations

from collections import Counter, defaultdict
from datetime import datetime, timedelta

from sqlalchemy.orm import Session

from .models import Event, Target


def build_digest(db: Session, target: Target, hours: int = 24) -> dict:
    since = datetime.utcnow() - timedelta(hours=hours)
    events = (
        db.query(Event)
        .filter(Event.target_id == target.id, Event.happened_at >= since)
        .order_by(Event.happened_at.desc())
        .all()
    )
    by_network: dict[str, list[dict]] = defaultdict(list)
    kinds: Counter = Counter()
    for e in events:
        by_network[e.network].append(
            {
                "id": e.id,
                "kind": e.kind,
                "title": e.title,
                "url": e.url,
                "happened_at": e.happened_at.isoformat(),
            }
        )
        kinds[f"{e.network}:{e.kind}"] += 1

    github_bullets = []
    for e in events:
        if e.network == "github":
            if e.kind in {"push", "pr", "issue", "repo"}:
                github_bullets.append(f"- {e.title}")
    github_summary = None
    if github_bullets:
        github_summary = (
            f"{len(github_bullets)} GitHub action(s) in the last {hours}h:\n"
            + "\n".join(github_bullets[:20])
        )

    return {
        "target": target.handle,
        "window_hours": hours,
        "generated_at": datetime.utcnow().isoformat(),
        "total_events": len(events),
        "by_network_counts": {k: len(v) for k, v in by_network.items()},
        "by_kind_counts": dict(kinds),
        "github_summary": github_summary,
        "events_by_network": by_network,
    }
