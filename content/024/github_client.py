import os
import httpx

GITHUB_API = "https://api.github.com"
TOKEN = os.getenv("GITHUB_TOKEN")


def _headers():
    h = {"Accept": "application/vnd.github+json", "User-Agent": "repo-pentefino-bot"}
    if TOKEN:
        h["Authorization"] = f"Bearer {TOKEN}"
    return h


async def _get(client: httpx.AsyncClient, url: str, params: dict | None = None):
    r = await client.get(url, headers=_headers(), params=params, timeout=30.0)
    r.raise_for_status()
    return r


async def _count_via_pagination(client: httpx.AsyncClient, url: str, params: dict) -> int:
    p = {**params, "per_page": 1}
    r = await _get(client, url, p)
    data = r.json()
    link = r.headers.get("Link", "")
    if 'rel="last"' in link:
        for part in link.split(","):
            if 'rel="last"' in part:
                last_url = part[part.find("<") + 1 : part.find(">")]
                from urllib.parse import urlparse, parse_qs
                q = parse_qs(urlparse(last_url).query)
                return int(q.get("page", ["1"])[0])
    return len(data) if isinstance(data, list) else 0


async def _search_count(client: httpx.AsyncClient, q: str) -> int:
    r = await _get(client, f"{GITHUB_API}/search/issues", {"q": q, "per_page": 1})
    return r.json().get("total_count", 0)


async def fetch_repo_metadata(owner: str, repo: str) -> dict:
    base = f"{GITHUB_API}/repos/{owner}/{repo}"
    async with httpx.AsyncClient(follow_redirects=True) as client:
        info = (await _get(client, base)).json()
        languages = (await _get(client, f"{base}/languages")).json()

        slug = info.get("full_name") or f"{owner}/{repo}"
        open_issues = await _search_count(client, f"repo:{slug} is:issue is:open")
        closed_issues = await _search_count(client, f"repo:{slug} is:issue is:closed")
        open_prs = await _search_count(client, f"repo:{slug} is:pr is:open")
        closed_prs = await _search_count(client, f"repo:{slug} is:pr is:closed")
        contributors_count = await _count_via_pagination(
            client, f"{base}/contributors", {"anon": "true"}
        )

        total_lines = sum(languages.values()) if languages else 0
        main_language = max(languages, key=languages.get) if languages else info.get("language")

        return {
            "full_name": info.get("full_name"),
            "description": info.get("description"),
            "main_language": main_language,
            "languages_bytes": languages,
            "total_bytes": total_lines,
            "stars": info.get("stargazers_count"),
            "forks": info.get("forks_count"),
            "watchers": info.get("subscribers_count"),
            "open_issues": open_issues,
            "closed_issues": closed_issues,
            "open_prs": open_prs,
            "closed_prs": closed_prs,
            "contributors": contributors_count,
            "default_branch": info.get("default_branch"),
            "created_at": info.get("created_at"),
            "updated_at": info.get("updated_at"),
            "pushed_at": info.get("pushed_at"),
            "archived": info.get("archived"),
            "license": (info.get("license") or {}).get("spdx_id"),
            "topics": info.get("topics", []),
        }
