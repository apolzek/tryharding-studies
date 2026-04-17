"""RedQueen launcher."""
from __future__ import annotations

import uvicorn

from backend.config import get_config


def main() -> None:
    cfg = get_config()
    uvicorn.run(
        "backend.main:app",
        host=cfg.server.host,
        port=cfg.server.port,
        log_level=cfg.server.log_level,
        reload=False,
    )


if __name__ == "__main__":
    main()
