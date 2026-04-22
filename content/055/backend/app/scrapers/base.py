from __future__ import annotations

import logging
from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import Iterable

import httpx
from dateutil import parser as date_parser

from ..config import HTTP_TIMEOUT_SECONDS, USER_AGENT

log = logging.getLogger("spy.scrapers")


@dataclass
class ScrapedEvent:
    network: str
    external_id: str
    title: str
    url: str
    happened_at: datetime
    kind: str = "post"
    body: str | None = None


class BaseScraper:
    network: str = ""

    def fetch(self, handle: str) -> Iterable[ScrapedEvent]:  # pragma: no cover - abstract
        raise NotImplementedError

    @staticmethod
    def http_get(url: str, *, accept: str | None = None) -> httpx.Response:
        # A browser-shaped Accept keeps sites like Reddit from 403-ing the
        # rss-specific Accept headers; the `accept` kwarg is advisory only.
        headers = {
            "User-Agent": USER_AGENT,
            "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
            "Accept-Language": "en-US,en;q=0.8",
        }
        with httpx.Client(timeout=HTTP_TIMEOUT_SECONDS, follow_redirects=True) as client:
            resp = client.get(url, headers=headers)
        resp.raise_for_status()
        return resp

    @staticmethod
    def parse_dt(value) -> datetime:
        if isinstance(value, datetime):
            dt = value
        elif isinstance(value, (int, float)):
            dt = datetime.fromtimestamp(value, tz=timezone.utc)
        else:
            dt = date_parser.parse(str(value))
        if dt.tzinfo is None:
            dt = dt.replace(tzinfo=timezone.utc)
        return dt.astimezone(timezone.utc).replace(tzinfo=None)
