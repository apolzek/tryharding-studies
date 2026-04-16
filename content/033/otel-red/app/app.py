import os
import random
import time
from flask import Flask, abort

app = Flask(__name__)

# Deterministic-ish per-endpoint latency profiles so p50/p95 answers are easy
# to validate when asking the MCP about latency.
PROFILES = {
    "fast":   (0.005, 0.020),   # 5-20 ms
    "medium": (0.050, 0.150),   # 50-150 ms
    "slow":   (0.400, 1.200),   # 400-1200 ms
}


def _latency(profile):
    lo, hi = PROFILES[profile]
    time.sleep(random.uniform(lo, hi))


@app.get("/")
def index():
    return "ok"


@app.get("/api/fast")
def fast():
    _latency("fast")
    return {"ok": True}


@app.get("/api/medium")
def medium():
    _latency("medium")
    return {"ok": True}


@app.get("/api/slow")
def slow():
    _latency("slow")
    return {"ok": True}


@app.get("/api/flaky")
def flaky():
    _latency("medium")
    if random.random() < 0.35:  # ~35% errors
        abort(500)
    return {"ok": True}


@app.get("/api/notfound")
def notfound():
    abort(404)


if __name__ == "__main__":
    port = int(os.environ.get("PORT", "8000"))
    app.run(host="0.0.0.0", port=port)
