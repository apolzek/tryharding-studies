from __future__ import annotations

import feedparser

from ..config import NITTER_BASE
from .base import BaseScraper, ScrapedEvent, log


class NitterScraper(BaseScraper):
    """Scrapes X/Twitter via a Nitter instance RSS feed. Set NITTER_BASE env
    to point at a healthy instance (defaults to nitter.net)."""

    network = "twitter"

    def fetch(self, handle: str):
        url = f"{NITTER_BASE.rstrip('/')}/{handle}/rss"
        try:
            resp = self.http_get(url, accept="application/rss+xml")
        except Exception as e:
            log.warning("nitter fetch failed for %s: %s", handle, e)
            return
        feed = feedparser.parse(resp.text)
        for entry in feed.entries:
            published = entry.get("published") or entry.get("updated")
            if not published:
                continue
            yield ScrapedEvent(
                network=self.network,
                external_id=entry.get("id") or entry.get("link"),
                title=(entry.get("title") or "tweet")[:200],
                url=entry.get("link"),
                happened_at=self.parse_dt(published),
                kind="tweet",
                body=(entry.get("summary") or "")[:2000] or None,
            )
