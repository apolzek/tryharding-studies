from __future__ import annotations

import feedparser

from .base import BaseScraper, ScrapedEvent, log


class YoutubeScraper(BaseScraper):
    """Resolves a handle to a channel via the public channel page, then reads
    the channel Atom feed. Nothing authenticated."""

    network = "youtube"

    def fetch(self, handle: str):
        channel_id = self._resolve_channel_id(handle)
        if not channel_id:
            log.info("youtube channel id not resolved for %s", handle)
            return
        url = f"https://www.youtube.com/feeds/videos.xml?channel_id={channel_id}"
        try:
            resp = self.http_get(url, accept="application/atom+xml")
        except Exception as e:
            log.warning("youtube feed fetch failed for %s: %s", handle, e)
            return
        feed = feedparser.parse(resp.text)
        for entry in feed.entries:
            published = entry.get("published") or entry.get("updated")
            if not published:
                continue
            yield ScrapedEvent(
                network=self.network,
                external_id=entry.get("id") or entry.get("link"),
                title=entry.get("title") or "YouTube upload",
                url=entry.get("link"),
                happened_at=self.parse_dt(published),
                kind="video",
                body=(entry.get("summary") or "")[:2000] or None,
            )

    def _resolve_channel_id(self, handle: str) -> str | None:
        handle = handle.lstrip("@")
        for candidate in (f"https://www.youtube.com/@{handle}", f"https://www.youtube.com/user/{handle}"):
            try:
                resp = self.http_get(candidate)
            except Exception:
                continue
            needle = 'channel_id='
            for marker in ('"channelId":"', '"externalId":"'):
                i = resp.text.find(marker)
                if i >= 0:
                    start = i + len(marker)
                    end = resp.text.find('"', start)
                    if end > start:
                        return resp.text[start:end]
            i = resp.text.find(needle)
            if i >= 0:
                start = i + len(needle)
                end = resp.text.find('"', start)
                return resp.text[start:end]
        return None
