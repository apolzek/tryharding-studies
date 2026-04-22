import os
from pathlib import Path

def _resolve_data_dir() -> Path:
    candidate = Path(os.getenv("DATA_DIR", "/data"))
    try:
        candidate.mkdir(parents=True, exist_ok=True)
        return candidate
    except PermissionError:
        fallback = Path(__file__).resolve().parent.parent / ".data"
        fallback.mkdir(parents=True, exist_ok=True)
        return fallback


DATA_DIR = _resolve_data_dir()

DATABASE_URL = os.getenv("DATABASE_URL", f"sqlite:///{DATA_DIR/'spy.db'}")

SCRAPE_INTERVAL_MINUTES = int(os.getenv("SCRAPE_INTERVAL_MINUTES", "15"))
HTTP_TIMEOUT_SECONDS = float(os.getenv("HTTP_TIMEOUT_SECONDS", "20"))
USER_AGENT = os.getenv(
    "SPY_USER_AGENT",
    "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36",
)

NITTER_BASE = os.getenv("NITTER_BASE", "https://nitter.net")
MASTODON_INSTANCE = os.getenv("MASTODON_INSTANCE", "mastodon.social")

SUPPORTED_NETWORKS = [
    "github",
    "devto",
    "medium",
    "reddit",
    "youtube",
    "hackernews",
    "mastodon",
    "twitter",
]
