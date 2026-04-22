"""Unit tests that stub httpx so the scrapers parse real fixture payloads
without touching the network."""
from __future__ import annotations

from datetime import datetime
from unittest.mock import patch

import httpx
import pytest

from app.scrapers.devto import DevtoScraper
from app.scrapers.github import GithubScraper
from app.scrapers.hackernews import HackerNewsScraper
from app.scrapers.reddit import RedditScraper


GITHUB_ATOM = """<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <id>tag:github.com,2008:/apolzek</id>
  <title>apolzek's activity</title>
  <entry>
    <id>tag:github.com,2008:Push/1</id>
    <title>apolzek pushed to main in apolzek/demo</title>
    <link href="https://github.com/apolzek/demo/commit/abc" />
    <published>2026-04-22T10:00:00Z</published>
    <summary>Commit message</summary>
  </entry>
  <entry>
    <id>tag:github.com,2008:PullRequestEvent/2</id>
    <title>apolzek opened a pull request in apolzek/demo</title>
    <link href="https://github.com/apolzek/demo/pull/1" />
    <published>2026-04-22T11:00:00Z</published>
  </entry>
</feed>
"""

DEVTO_RSS = """<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"><channel>
  <title>dev.to feed</title>
  <item>
    <title>Hello from dev.to</title>
    <link>https://dev.to/apolzek/hello</link>
    <guid>https://dev.to/apolzek/hello</guid>
    <pubDate>Tue, 22 Apr 2026 09:00:00 GMT</pubDate>
    <description>summary</description>
  </item>
</channel></rss>
"""

REDDIT_RSS = """<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <entry>
    <id>t3_abc</id>
    <title>Post by /u/apolzek</title>
    <link href="https://old.reddit.com/r/x/abc" />
    <updated>2026-04-22T08:00:00+00:00</updated>
    <content>body</content>
  </entry>
</feed>
"""

HN_HTML = """<html><body><table>
  <tr class="athing" id="12345">
    <td><span class="titleline"><a href="https://example.com">Cool thing</a></span></td>
  </tr>
  <tr>
    <td><span class="age" title="2026-04-22T07:00:00"><a>1 hour ago</a></span></td>
  </tr>
</table></body></html>
"""


class _StubResponse:
    def __init__(self, text: str):
        self.text = text
        self.status_code = 200

    def raise_for_status(self):
        return None


class _StubClient:
    def __init__(self, text: str):
        self._text = text

    def __enter__(self):
        return self

    def __exit__(self, *a):
        return False

    def get(self, *a, **kw):
        return _StubResponse(self._text)


def _patch_http(payload: str):
    return patch.object(httpx, "Client", lambda *a, **kw: _StubClient(payload))


def test_github_atom_parses():
    with _patch_http(GITHUB_ATOM), patch.object(GithubScraper, "_from_profile", return_value=None):
        events = list(GithubScraper().fetch("apolzek"))
    assert len(events) == 2
    kinds = {e.kind for e in events}
    assert "push" in kinds
    assert "pr" in kinds
    assert all(isinstance(e.happened_at, datetime) for e in events)


def test_devto_rss_parses():
    with _patch_http(DEVTO_RSS):
        events = list(DevtoScraper().fetch("apolzek"))
    assert len(events) == 1
    assert events[0].network == "devto"
    assert events[0].url.endswith("/hello")


def test_reddit_feed_parses():
    with _patch_http(REDDIT_RSS):
        events = list(RedditScraper().fetch("apolzek"))
    assert len(events) == 1
    assert events[0].network == "reddit"


def test_hn_html_parses():
    with _patch_http(HN_HTML):
        events = list(HackerNewsScraper().fetch("apolzek"))
    assert len(events) == 1
    assert events[0].network == "hackernews"
    assert events[0].title == "Cool thing"


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
