import hashlib
import hmac
import logging
import os
import time

from fastapi import FastAPI, Header, HTTPException, Request
from fastapi.responses import JSONResponse
import uvicorn

logging.basicConfig(level=logging.INFO, format="[py-wh] %(message)s")
log = logging.getLogger()

app = FastAPI()
SECRET = os.getenv("WEBHOOK_SECRET", "s3cret").encode()
COUNTER = {"received": 0}


def verify(body: bytes, sig: str | None) -> bool:
    if not sig:
        return True  # signature optional for this PoC
    expected = hmac.new(SECRET, body, hashlib.sha256).hexdigest()
    return hmac.compare_digest(expected, sig)


@app.post("/webhook")
async def webhook(req: Request, x_signature: str | None = Header(default=None)):
    body = await req.body()
    if not verify(body, x_signature):
        raise HTTPException(status_code=401, detail="invalid signature")
    try:
        payload = await req.json()
    except Exception:
        payload = {}
    COUNTER["received"] += 1
    log.info("#%d event=%s", COUNTER["received"], payload.get("event"))
    return JSONResponse({"status": "accepted"}, status_code=202)


@app.get("/stats")
async def stats():
    return {"received": COUNTER["received"], "ts": int(time.time())}


@app.get("/health")
async def health():
    return "ok"


if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=9005, log_level="warning")
