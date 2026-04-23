"""In-process chaos injector — togglable via POST /chaos/inject.

Real systems are almost always better with a panic button like this
for on-call drills. Here we do two simple knobs:
- `latency_ms`: added sleep on every request.
- `error_rate`: probability of returning a 503.
"""
from __future__ import annotations

import asyncio
import random

from starlette.middleware.base import BaseHTTPMiddleware
from starlette.responses import JSONResponse


class ChaosState:
    def __init__(self):
        self.latency_ms = 0
        self.error_rate = 0.0

    def to_dict(self):
        return {"latency_ms": self.latency_ms, "error_rate": self.error_rate}


class ChaosMiddleware(BaseHTTPMiddleware):
    def __init__(self, app, state: ChaosState):
        super().__init__(app)
        self.state = state

    async def dispatch(self, request, call_next):
        if request.url.path.startswith("/chaos") or request.url.path == "/health":
            return await call_next(request)
        if self.state.latency_ms:
            await asyncio.sleep(self.state.latency_ms / 1000)
        if self.state.error_rate and random.random() < self.state.error_rate:
            return JSONResponse({"error": "chaos: injected failure"}, status_code=503)
        return await call_next(request)
