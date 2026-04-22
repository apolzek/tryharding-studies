from __future__ import annotations

from .base import BaseScraper, ScrapedEvent
from .devto import DevtoScraper
from .github import GithubScraper
from .hackernews import HackerNewsScraper
from .mastodon import MastodonScraper
from .medium import MediumScraper
from .nitter import NitterScraper
from .reddit import RedditScraper
from .youtube import YoutubeScraper

REGISTRY: dict[str, BaseScraper] = {
    s.network: s
    for s in [
        GithubScraper(),
        DevtoScraper(),
        MediumScraper(),
        RedditScraper(),
        YoutubeScraper(),
        HackerNewsScraper(),
        MastodonScraper(),
        NitterScraper(),
    ]
}


def scrape_target(network: str, handle: str) -> list[ScrapedEvent]:
    scraper = REGISTRY.get(network)
    if scraper is None:
        return []
    return list(scraper.fetch(handle))
