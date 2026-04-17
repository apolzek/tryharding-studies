"""Cliente WebSocket simples para teste manual.

Uso:
  python ws_client.py ws://localhost:8001/ws
"""
import asyncio
import sys

import websockets


async def main(url: str) -> None:
    async with websockets.connect(url) as ws:
        print(f"connected to {url}")
        send_task = asyncio.create_task(sender(ws))
        recv_task = asyncio.create_task(receiver(ws))
        done, pending = await asyncio.wait(
            [send_task, recv_task], return_when=asyncio.FIRST_COMPLETED
        )
        for t in pending:
            t.cancel()


async def sender(ws) -> None:
    loop = asyncio.get_event_loop()
    while True:
        line = await loop.run_in_executor(None, input, "> ")
        if not line:
            continue
        await ws.send(line)


async def receiver(ws) -> None:
    async for msg in ws:
        print(f"\n< {msg}\n> ", end="", flush=True)


if __name__ == "__main__":
    url = sys.argv[1] if len(sys.argv) > 1 else "ws://localhost:8001/ws"
    asyncio.run(main(url))
