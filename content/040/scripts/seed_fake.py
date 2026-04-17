#!/usr/bin/env python3
"""
Seed the backend with 24h of synthetic samples for two users, so the UI can be
exercised without needing root + eBPF. Real workflow is: agent -> backend; this
script shortcuts straight to ingestion for smoke-testing.
"""
from __future__ import annotations

import argparse
import json
import random
import time
import urllib.request

def post(url: str, token: str, payload: dict) -> None:
    req = urllib.request.Request(
        url,
        data=json.dumps(payload).encode(),
        headers={"Content-Type": "application/json",
                 "Authorization": f"Bearer {token}"},
        method="POST",
    )
    with urllib.request.urlopen(req, timeout=5) as resp:
        assert resp.status < 300, f"ingest failed: {resp.status}"

def gen_sample(user: str, uid: int, machine_id: str, host: str,
               t_end: float, interval: float,
               active: bool, network_only: bool = False) -> dict:
    # Shape events so classifier can see realistic volumes.
    if active:
        keys = random.randint(40, 180) * int(interval / 60 if interval > 60 else 1)
        clicks = random.randint(3, 15)
        moves = random.randint(150, 800)
        scrolls = random.randint(0, 20)
        rx = random.randint(50_000, 2_000_000)
        tx = random.randint(20_000, 600_000)
    elif network_only:
        keys = 0
        clicks = random.randint(0, 1)
        moves = random.randint(0, 10)
        scrolls = 0
        rx = random.randint(400_000, 5_000_000)
        tx = random.randint(100_000, 2_000_000)
    else:
        # idle / tiny activity
        keys = random.randint(0, 3)
        clicks = random.randint(0, 1)
        moves = random.randint(0, 30)
        scrolls = 0
        rx = random.randint(0, 20_000)
        tx = random.randint(0, 5_000)

    return {
        "agent_version": "1.0.0-fake",
        "host": host,
        "machine_id": machine_id,
        "kernel": "6.8.0-fake",
        "os": "Ubuntu 24.04",
        "session_user": user,
        "session_uid": uid,
        "window_start": t_end - interval,
        "window_end": t_end,
        "interval_sec": interval,
        "keys_pressed": keys,
        "keys_released": max(0, keys - 1),
        "clicks": clicks,
        "mouse_moves": moves,
        "scrolls": scrolls,
        "rx_bytes": rx,
        "tx_bytes": tx,
        "net_calls": (rx + tx) // 1000 + 1,
        "per_user_net": [
            {"uid": uid, "user": user,
             "rx_bytes": rx, "tx_bytes": tx,
             "rx_calls": (rx // 1500) or 1, "tx_calls": (tx // 1500) or 1},
        ],
        "load1": round(random.uniform(0.1, 2.0), 2),
        "load5": round(random.uniform(0.1, 1.5), 2),
        "load15": round(random.uniform(0.1, 1.2), 2),
        "mem_used_pct": round(random.uniform(30, 75), 2),
        "cpu_used_pct": round(random.uniform(5, 60) if active else random.uniform(1, 15), 2),
        "num_procs": random.randint(200, 450),
    }

def seed_user(backend: str, token: str, user: str, uid: int,
              machine_id: str, host: str,
              hours: float, productive_hours: float,
              interval: float) -> int:
    """Produce `hours` of history ending now, with `productive_hours` of active
    time + some 'network' time so productivity > target."""
    url = backend.rstrip("/") + "/api/v1/ingest"
    now = time.time()
    start = now - hours * 3600
    total_windows = int((hours * 3600) / interval)
    productive_windows = int((productive_hours * 3600) / interval)
    # pick which windows are productive (clumped, not random — more realistic)
    is_productive = [False] * total_windows
    i = 0
    remaining = productive_windows
    while remaining > 0 and i < total_windows:
        # clump of 15-60 minutes productive, then idle gap
        clump_len = random.randint(int(15*60/interval), int(60*60/interval))
        clump_len = min(clump_len, remaining, total_windows - i)
        for k in range(clump_len):
            is_productive[i + k] = True
        i += clump_len
        gap = random.randint(int(5*60/interval), int(20*60/interval))
        i += gap
        remaining -= clump_len

    sent = 0
    for w in range(total_windows):
        t_end = start + (w + 1) * interval
        kind = "active" if is_productive[w] else ("network" if random.random() < 0.1 else "idle")
        sample = gen_sample(
            user, uid, machine_id, host, t_end, interval,
            active=(kind == "active"),
            network_only=(kind == "network"),
        )
        post(url, token, sample)
        sent += 1
    return sent

def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--backend", default="http://127.0.0.1:8080")
    ap.add_argument("--token", default="dev-token")
    ap.add_argument("--interval", type=float, default=60.0,
                    help="seconds per sample (default 60)")
    ap.add_argument("--hours", type=float, default=24.0,
                    help="how many hours of history to seed")
    args = ap.parse_args()

    users = [
        ("alice", 1001, "11111111aaaabbbbccccdddd", "ws-alice", 9.5),
        ("bob",   1002, "22222222aaaabbbbccccdddd", "ws-bob",   4.0),
        ("carol", 1003, "33333333aaaabbbbccccdddd", "ws-carol", 7.2),
    ]
    total = 0
    for user, uid, mid, host, productive in users:
        n = seed_user(args.backend, args.token, user, uid, mid, host,
                      args.hours, productive, args.interval)
        print(f"seeded {n:>5} samples for {user:<6} productive={productive}h")
        total += n
    print(f"done. total samples: {total}")
    return 0

if __name__ == "__main__":
    raise SystemExit(main())
