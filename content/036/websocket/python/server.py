import asyncio
import json
import logging
import signal

import websockets

logging.basicConfig(level=logging.INFO, format="[py-ws] %(message)s")
log = logging.getLogger()

CLIENTS: set[websockets.WebSocketServerProtocol] = set()


async def broadcast(message: str) -> None:
    if not CLIENTS:
        return
    await asyncio.gather(
        *(c.send(message) for c in CLIENTS),
        return_exceptions=True,
    )


async def handler(ws: websockets.WebSocketServerProtocol) -> None:
    CLIENTS.add(ws)
    log.info("client connected (total=%d)", len(CLIENTS))
    try:
        async for msg in ws:
            log.info("received: %s", msg)
            await broadcast(msg)
    except websockets.ConnectionClosed:
        pass
    finally:
        CLIENTS.discard(ws)
        log.info("client disconnected (total=%d)", len(CLIENTS))


async def main() -> None:
    stop = asyncio.get_event_loop().create_future()
    for sig in (signal.SIGINT, signal.SIGTERM):
        asyncio.get_event_loop().add_signal_handler(sig, stop.set_result, None)

    async with websockets.serve(
        handler,
        "0.0.0.0",
        8005,
        ping_interval=20,
        ping_timeout=20,
        max_size=1 << 20,
    ):
        log.info("listening on :8005")
        await stop


if __name__ == "__main__":
    asyncio.run(main())
