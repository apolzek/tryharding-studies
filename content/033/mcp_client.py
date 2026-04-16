#!/usr/bin/env python3
"""
Minimal MCP client for stdio transport used to exercise each server.

Usage:
    ./mcp_client.py <cmd> [args...]

Sends initialize + notifications/initialized + tools/list and prints the
tool list. Then for each argument passed via --call NAME JSON it calls
the tool and prints the result.

Example:
    ./mcp_client.py --call list_metrics '{}' -- \
        docker run --rm -i --network host \
        -e PROMETHEUS_URL=http://127.0.0.1:9090 \
        ghcr.io/pab1it0/prometheus-mcp-server:latest
"""

import json
import subprocess
import sys
import threading
import queue
import time


def reader(stream, q):
    for line in iter(stream.readline, b""):
        q.put(line.decode("utf-8", errors="replace"))
    q.put(None)


def run(cmd, calls, timeout=25):
    proc = subprocess.Popen(
        cmd,
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        bufsize=0,
    )
    out_q = queue.Queue()
    err_q = queue.Queue()
    threading.Thread(target=reader, args=(proc.stdout, out_q), daemon=True).start()
    threading.Thread(target=reader, args=(proc.stderr, err_q), daemon=True).start()

    def send(obj):
        data = (json.dumps(obj) + "\n").encode("utf-8")
        proc.stdin.write(data)
        proc.stdin.flush()

    def recv(want_id, deadline):
        while time.time() < deadline:
            try:
                line = out_q.get(timeout=0.5)
            except queue.Empty:
                if proc.poll() is not None:
                    break
                continue
            if line is None:
                break
            line = line.strip()
            if not line:
                continue
            try:
                msg = json.loads(line)
            except Exception:
                continue
            if msg.get("id") == want_id:
                return msg
        return None

    results = {"init": None, "tools": [], "calls": []}
    send({
        "jsonrpc": "2.0", "id": 1, "method": "initialize",
        "params": {
            "protocolVersion": "2025-03-26",
            "capabilities": {},
            "clientInfo": {"name": "tryharding-test", "version": "0.1.0"},
        },
    })
    deadline = time.time() + timeout
    init = recv(1, deadline)
    results["init"] = init

    send({"jsonrpc": "2.0", "method": "notifications/initialized", "params": {}})

    send({"jsonrpc": "2.0", "id": 2, "method": "tools/list", "params": {}})
    tools_msg = recv(2, time.time() + timeout)
    if tools_msg and "result" in tools_msg:
        results["tools"] = [t.get("name") for t in tools_msg["result"].get("tools", [])]

    for i, (name, args) in enumerate(calls, start=3):
        send({
            "jsonrpc": "2.0", "id": i, "method": "tools/call",
            "params": {"name": name, "arguments": args},
        })
        resp = recv(i, time.time() + timeout)
        results["calls"].append({"name": name, "args": args, "resp": resp})

    try:
        proc.stdin.close()
    except Exception:
        pass
    try:
        proc.wait(timeout=3)
    except Exception:
        proc.kill()

    stderr_lines = []
    while True:
        try:
            line = err_q.get_nowait()
        except queue.Empty:
            break
        if line is None:
            break
        stderr_lines.append(line)
    results["stderr_tail"] = "".join(stderr_lines[-30:])
    return results


def main(argv):
    calls = []
    cmd = []
    i = 0
    while i < len(argv):
        a = argv[i]
        if a == "--call":
            calls.append((argv[i + 1], json.loads(argv[i + 2])))
            i += 3
        elif a == "--":
            cmd = argv[i + 1:]
            break
        else:
            cmd = argv[i:]
            break
    if not cmd:
        print("usage: mcp_client.py [--call NAME JSON]... -- cmd [args...]", file=sys.stderr)
        return 2
    r = run(cmd, calls)
    print(json.dumps({
        "initOk": bool(r["init"] and "result" in r["init"]),
        "serverInfo": (r["init"] or {}).get("result", {}).get("serverInfo"),
        "toolCount": len(r["tools"]),
        "tools": r["tools"],
        "calls": [
            {
                "name": c["name"],
                "ok": bool(c["resp"] and "result" in c["resp"]),
                "preview": _preview(c["resp"]),
            }
            for c in r["calls"]
        ],
        "stderrTail": r["stderr_tail"][-800:],
    }, indent=2))
    return 0


def _preview(resp):
    if not resp:
        return None
    res = resp.get("result")
    if res is None:
        return resp.get("error")
    content = res.get("content") if isinstance(res, dict) else None
    if isinstance(content, list) and content:
        first = content[0]
        if isinstance(first, dict) and "text" in first:
            text = first["text"]
            return text[:300] + ("..." if len(text) > 300 else "")
    return str(res)[:300]


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
