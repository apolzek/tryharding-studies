from __future__ import annotations

import feedparser

from .base import BaseScraper, ScrapedEvent, log


class RedditScraper(BaseScraper):
    """Uses the public old.reddit.com RSS endpoint (no login, no API token)."""

    network = "reddit"

    def fetch(self, handle: str):
        url = f"https://old.reddit.com/user/{handle}/.rss"
        try:
            resp = self.http_get(url, accept="application/rss+xml")
        except Exception as e:
            log.warning("reddit fetch failed for %s: %s", handle, e)
            return
        feed = feedparser.parse(resp.text)
        for entry in feed.entries:
            published = entry.get("published") or entry.get("updated")
            if not published:
                continue
            link = entry.get("link")
            title = entry.get("title") or "reddit activity"
            kind = "comment" if title.lower().startswith("comment") else "post"
            yield ScrapedEvent(
                network=self.network,
                external_id=entry.get("id") or link,
                title=title,
                url=link,
                happened_at=self.parse_dt(published),
                kind=kind,
                body=(entry.get("summary") or "")[:2000] or None,
            )
