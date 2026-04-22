from __future__ import annotations

import feedparser

from ..config import MASTODON_INSTANCE
from .base import BaseScraper, ScrapedEvent, log


class MastodonScraper(BaseScraper):
    """Any Mastodon instance exposes public toots as an RSS feed at /@user.rss.
    Users can pass either `username` (uses default instance) or `username@server`."""

    network = "mastodon"

    def fetch(self, handle: str):
        if "@" in handle:
            user, _, instance = handle.partition("@")
        else:
            user, instance = handle, MASTODON_INSTANCE
        url = f"https://{instance}/@{user}.rss"
        try:
            resp = self.http_get(url, accept="application/rss+xml")
        except Exception as e:
            log.warning("mastodon fetch failed for %s@%s: %s", user, instance, e)
            return
        feed = feedparser.parse(resp.text)
        for entry in feed.entries:
            published = entry.get("published") or entry.get("updated")
            if not published:
                continue
            yield ScrapedEvent(
                network=self.network,
                external_id=entry.get("id") or entry.get("link"),
                title=(entry.get("title") or entry.get("summary") or "toot")[:200],
                url=entry.get("link"),
                happened_at=self.parse_dt(published),
                kind="toot",
                body=(entry.get("summary") or "")[:2000] or None,
            )
