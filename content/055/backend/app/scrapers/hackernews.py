from __future__ import annotations

from datetime import datetime

from bs4 import BeautifulSoup

from .base import BaseScraper, ScrapedEvent, log


class HackerNewsScraper(BaseScraper):
    """Scrapes the public HN user submissions/comments pages (news.ycombinator.com/submitted?id=user)."""

    network = "hackernews"

    def fetch(self, handle: str):
        yield from self._fetch_submissions(handle)

    def _fetch_submissions(self, handle: str):
        url = f"https://news.ycombinator.com/submitted?id={handle}"
        try:
            resp = self.http_get(url)
        except Exception as e:
            log.warning("hn fetch failed for %s: %s", handle, e)
            return
        soup = BeautifulSoup(resp.text, "html.parser")
        for row in soup.select("tr.athing"):
            title_el = row.select_one("span.titleline > a")
            if not title_el:
                continue
            title = title_el.get_text(strip=True)
            link = title_el.get("href") or ""
            hn_id = row.get("id") or link
            subtext = row.find_next_sibling("tr")
            ts_attr = None
            if subtext is not None:
                age_el = subtext.select_one("span.age")
                if age_el is not None:
                    ts_attr = age_el.get("title")
            try:
                happened = self.parse_dt(ts_attr) if ts_attr else datetime.utcnow()
            except Exception:
                happened = datetime.utcnow()
            if link.startswith("item?id="):
                link = f"https://news.ycombinator.com/{link}"
            yield ScrapedEvent(
                network=self.network,
                external_id=f"hn:{hn_id}",
                title=title,
                url=link,
                happened_at=happened,
                kind="submission",
            )
