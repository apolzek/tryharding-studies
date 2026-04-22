from __future__ import annotations

import feedparser

from .base import BaseScraper, ScrapedEvent, log


class MediumScraper(BaseScraper):
    network = "medium"

    def fetch(self, handle: str):
        url = f"https://medium.com/feed/@{handle}"
        try:
            resp = self.http_get(url, accept="application/rss+xml")
        except Exception as e:
            log.warning("medium fetch failed for %s: %s", handle, e)
            return
        feed = feedparser.parse(resp.text)
        for entry in feed.entries:
            published = entry.get("published") or entry.get("updated")
            if not published:
                continue
            link = entry.get("link")
            yield ScrapedEvent(
                network=self.network,
                external_id=entry.get("id") or link,
                title=entry.get("title") or "Medium post",
                url=link,
                happened_at=self.parse_dt(published),
                kind="post",
                body=(entry.get("summary") or "")[:2000] or None,
            )
