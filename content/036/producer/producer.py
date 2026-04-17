"""
Gerador de eventos: envia webhooks HTTP para todos os receptores e
opcionalmente publica mensagens nos servidores WebSocket via cliente.

Uso:
  python producer.py webhook            # dispara 1 burst para todos WH
  python producer.py webhook --rps 50   # 50 req/s durante --duration s
  python producer.py ws                 # publica 1 msg em cada WS
"""
import argparse
import asyncio
import hashlib
import hmac
import json
import os
import time
from typing import Iterable

import httpx
import websockets

WEBHOOK_TARGETS = [
    ("go",     os.getenv("WH_GO",     "http://localhost:9001/webhook")),
    ("java",   os.getenv("WH_JAVA",   "http://localhost:9002/webhook")),
    ("dotnet", os.getenv("WH_DOTNET", "http://localhost:9003/webhook")),
    ("js",     os.getenv("WH_JS",     "http://localhost:9004/webhook")),
    ("python", os.getenv("WH_PY",     "http://localhost:9005/webhook")),
]

WS_TARGETS = [
    ("go",     os.getenv("WS_GO",     "ws://localhost:8001/ws")),
    ("java",   os.getenv("WS_JAVA",   "ws://localhost:8002/ws")),
    ("dotnet", os.getenv("WS_DOTNET", "ws://localhost:8003/ws")),
    ("js",     os.getenv("WS_JS",     "ws://localhost:8004/ws")),
    ("python", os.getenv("WS_PY",     "ws://localhost:8005")),
]

SECRET = os.getenv("WEBHOOK_SECRET", "s3cret").encode()


def sign(body: bytes) -> str:
    return hmac.new(SECRET, body, hashlib.sha256).hexdigest()


def build_payload(i: int) -> bytes:
    return json.dumps({
        "event": "order.created",
        "id": i,
        "amount": 100 + i,
        "ts": int(time.time()),
    }).encode()


async def send_once(client: httpx.AsyncClient, name: str, url: str, i: int) -> tuple[str, int, float]:
    body = build_payload(i)
    headers = {"Content-Type": "application/json", "X-Signature": sign(body)}
    t0 = time.perf_counter()
    try:
        r = await client.post(url, content=body, headers=headers, timeout=5.0)
        return name, r.status_code, (time.perf_counter() - t0) * 1000
    except Exception as e:
        return name, -1, (time.perf_counter() - t0) * 1000


async def burst(rps: int, duration: float, targets: Iterable[tuple[str, str]]) -> None:
    targets = list(targets)
    async with httpx.AsyncClient(http2=False) as client:
        end = time.time() + duration
        i = 0
        while time.time() < end:
            batch_start = time.time()
            coros = [send_once(client, n, u, i + k) for k, (n, u) in enumerate(targets) for _ in range(max(1, rps // max(1, len(targets))))]
            results = await asyncio.gather(*coros)
            i += len(results)
            failed = sum(1 for _, s, _ in results if s < 200 or s >= 300)
            avg = sum(l for _, _, l in results) / max(1, len(results))
            print(f"[producer] sent={len(results)} failed={failed} avg_latency={avg:.1f}ms")
            elapsed = time.time() - batch_start
            if elapsed < 1:
                await asyncio.sleep(1 - elapsed)


async def ws_publish() -> None:
    for name, url in WS_TARGETS:
        msg = json.dumps({"event": "notify", "to": name, "ts": int(time.time())})
        try:
            async with websockets.connect(url, open_timeout=5) as ws:
                await ws.send(msg)
                print(f"[producer-ws] {name}: sent {msg}")
        except Exception as e:
            print(f"[producer-ws] {name}: ERROR {e}")


def main() -> None:
    p = argparse.ArgumentParser()
    p.add_argument("mode", choices=["webhook", "ws"])
    p.add_argument("--rps", type=int, default=10, help="requests per second per target (webhook mode)")
    p.add_argument("--duration", type=float, default=5.0)
    args = p.parse_args()

    if args.mode == "webhook":
        asyncio.run(burst(args.rps, args.duration, WEBHOOK_TARGETS))
    else:
        asyncio.run(ws_publish())


if __name__ == "__main__":
    main()
