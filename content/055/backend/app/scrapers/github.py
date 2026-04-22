from __future__ import annotations

import feedparser
from bs4 import BeautifulSoup

from .base import BaseScraper, ScrapedEvent, log


class GithubScraper(BaseScraper):
    """Scrapes GitHub public activity via the public Atom feed at /{user}.atom
    plus the profile HTML for bio/contribution summary. No API token used.
    """

    network = "github"

    def fetch(self, handle: str):
        events = list(self._from_atom(handle))
        profile = self._from_profile(handle)
        if profile is not None:
            events.append(profile)
        events.extend(self._from_repos(handle))
        return events

    def _from_repos(self, handle: str):
        url = f"https://github.com/{handle}?tab=repositories"
        try:
            resp = self.http_get(url)
        except Exception as e:
            log.warning("github repos fetch failed for %s: %s", handle, e)
            return
        soup = BeautifulSoup(resp.text, "html.parser")
        for li in soup.select('li[itemprop="owns"]'):
            a = li.select_one("h3 a")
            if not a:
                continue
            name = a.get_text(strip=True)
            href = a.get("href") or ""
            rt = li.select_one("relative-time")
            dt_attr = rt.get("datetime") if rt else None
            if not dt_attr:
                continue
            desc_el = li.select_one('p[itemprop="description"], p.col-9, p.pinned-item-desc')
            lang_el = li.select_one('span[itemprop="programmingLanguage"]')
            desc = desc_el.get_text(" ", strip=True) if desc_el else ""
            lang = lang_el.get_text(strip=True) if lang_el else ""
            body = " | ".join(x for x in [lang, desc] if x)
            yield ScrapedEvent(
                network=self.network,
                external_id=f"repo:{href}",
                title=f"{handle}/{name}",
                url=f"https://github.com{href}",
                happened_at=self.parse_dt(dt_attr),
                kind="repo",
                body=body or None,
            )

    def _from_atom(self, handle: str):
        url = f"https://github.com/{handle}.atom"
        try:
            resp = self.http_get(url, accept="application/atom+xml")
        except Exception as e:
            log.warning("github atom fetch failed for %s: %s", handle, e)
            return
        feed = feedparser.parse(resp.text)
        for entry in feed.entries:
            title = entry.get("title") or "GitHub activity"
            link = entry.get("link") or f"https://github.com/{handle}"
            eid = entry.get("id") or link
            published = entry.get("published") or entry.get("updated")
            if not published:
                continue
            kind = self._classify(title)
            body = entry.get("summary") or ""
            yield ScrapedEvent(
                network=self.network,
                external_id=eid,
                title=title,
                url=link,
                happened_at=self.parse_dt(published),
                kind=kind,
                body=body[:2000] if body else None,
            )

    def _from_profile(self, handle: str):
        url = f"https://github.com/{handle}"
        try:
            resp = self.http_get(url)
        except Exception as e:
            log.warning("github profile fetch failed for %s: %s", handle, e)
            return None
        soup = BeautifulSoup(resp.text, "html.parser")
        bio_el = soup.select_one("div.user-profile-bio")
        bio = bio_el.get_text(" ", strip=True) if bio_el else ""
        contrib_el = soup.select_one("div.js-yearly-contributions h2")
        contrib_text = contrib_el.get_text(" ", strip=True) if contrib_el else ""
        if not (bio or contrib_text):
            return None
        from datetime import datetime

        today = datetime.utcnow().strftime("%Y-%m-%d")
        body = " | ".join(x for x in [contrib_text, bio] if x)
        return ScrapedEvent(
            network=self.network,
            external_id=f"profile:{handle}:{today}",
            title=f"GitHub profile snapshot — {handle}",
            url=url,
            happened_at=self.parse_dt(datetime.utcnow()),
            kind="profile",
            body=body,
        )

    @staticmethod
    def _classify(title: str) -> str:
        t = title.lower()
        if "pushed to" in t or "created a branch" in t:
            return "push"
        if "opened a pull request" in t or "merged" in t:
            return "pr"
        if "opened an issue" in t or "closed an issue" in t:
            return "issue"
        if "starred" in t:
            return "star"
        if "forked" in t:
            return "fork"
        if "created a repository" in t:
            return "repo"
        if "made" in t and "comment" in t:
            return "comment"
        return "activity"
