#!/usr/bin/env python3
"""
Tiny HTTP backend for the MCP playground. No third-party deps.

- GET  /              index.html
- GET  /api/servers   -> [{id, label}]
- POST /api/tools     body {server} -> {tools: [...]}
- POST /api/call      body {server, tool, arguments} -> {response}

Each call spawns a fresh `docker run -i` of the target MCP, drives the
JSON-RPC handshake (initialize / notifications/initialized / tools/*), and
tears the container down. ~500ms overhead per call, but keeps the backend
stateless.
"""

import http.server
import json
import os
import queue
import subprocess
import sys
import threading
import time

NET = os.environ.get("MCP_NET", "mcp033-otel-red_obs")
PROM = os.environ.get("MCP_PROM", "http://prometheus:9090")
VM = os.environ.get("MCP_VM", "http://victoriametrics:8428")
GF = os.environ.get("MCP_GF", "http://grafana:3000")

SERVERS = {
    "pab1it0": {
        "label": "pab1it0/prometheus-mcp-server — Python, 6 tools",
        "cmd": [
            "docker", "run", "--rm", "-i", "--network", NET,
            "-e", f"PROMETHEUS_URL={PROM}",
            "ghcr.io/pab1it0/prometheus-mcp-server:latest",
        ],
    },
    "tjhop": {
        "label": "tjhop/prometheus-mcp-server — Go, 28 tools (full Prom API + docs)",
        "cmd": [
            "docker", "run", "--rm", "-i", "--network", NET,
            "ghcr.io/tjhop/prometheus-mcp-server:latest",
            f"--prometheus.url={PROM}",
            "--mcp.transport=stdio",
        ],
    },
    "giantswarm": {
        "label": "giantswarm/mcp-prometheus — Go, 18 read-only tools (alerts)",
        "cmd": [
            "docker", "run", "--rm", "-i", "--network", NET,
            "-e", f"PROMETHEUS_URL={PROM}",
            "local/mcp-prometheus:latest", "serve", "--transport=stdio",
        ],
    },
    "vm": {
        "label": "VictoriaMetrics/mcp-victoriametrics — Go, 16 tools",
        "cmd": [
            "docker", "run", "--rm", "-i", "--network", NET,
            "-e", f"VM_INSTANCE_ENTRYPOINT={VM}",
            "-e", "VM_INSTANCE_TYPE=single",
            "ghcr.io/victoriametrics/mcp-victoriametrics:latest", "--mode=stdio",
        ],
    },
    "grafana": {
        "label": "grafana/mcp-grafana — Go, 50 tools",
        "cmd": [
            "docker", "run", "--rm", "-i", "--network", NET,
            "-e", f"GRAFANA_URL={GF}",
            "-e", "GRAFANA_USERNAME=admin",
            "-e", "GRAFANA_PASSWORD=admin",
            "grafana/mcp-grafana:latest", "-t", "stdio",
        ],
    },
}


def run_mcp(cmd, actions, timeout=25):
    proc = subprocess.Popen(
        cmd, stdin=subprocess.PIPE, stdout=subprocess.PIPE,
        stderr=subprocess.PIPE, bufsize=0,
    )
    out_q = queue.Queue()

    def reader():
        for line in iter(proc.stdout.readline, b""):
            out_q.put(line.decode("utf-8", "replace"))
        out_q.put(None)

    threading.Thread(target=reader, daemon=True).start()

    def send(obj):
        proc.stdin.write((json.dumps(obj) + "\n").encode())
        proc.stdin.flush()

    def recv(want_id, deadline):
        while time.time() < deadline:
            try:
                line = out_q.get(timeout=0.3)
            except queue.Empty:
                if proc.poll() is not None:
                    return None
                continue
            if line is None:
                return None
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

    send({
        "jsonrpc": "2.0", "id": 1, "method": "initialize",
        "params": {
            "protocolVersion": "2025-03-26",
            "capabilities": {},
            "clientInfo": {"name": "mcp-ui", "version": "0.1"},
        },
    })
    init = recv(1, time.time() + timeout)
    send({"jsonrpc": "2.0", "method": "notifications/initialized", "params": {}})

    responses = []
    for i, act in enumerate(actions, start=2):
        act["id"] = i
        send(act)
        responses.append(recv(i, time.time() + timeout))

    try:
        proc.stdin.close()
    except Exception:
        pass
    try:
        proc.wait(timeout=2)
    except Exception:
        proc.kill()

    return {"init": init, "responses": responses}


class Handler(http.server.BaseHTTPRequestHandler):
    def log_message(self, *a, **k):
        pass

    def _write(self, code, body, content_type="application/json"):
        if isinstance(body, str):
            body = body.encode("utf-8")
        self.send_response(code)
        self.send_header("Content-Type", content_type)
        self.send_header("Access-Control-Allow-Origin", "*")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_GET(self):
        if self.path in ("/", "/index.html"):
            path = os.path.join(os.path.dirname(__file__), "index.html")
            with open(path, "rb") as f:
                self._write(200, f.read(), "text/html; charset=utf-8")
            return
        if self.path == "/api/servers":
            payload = [
                {"id": k, "label": v["label"]} for k, v in SERVERS.items()
            ]
            self._write(200, json.dumps(payload))
            return
        self._write(404, '{"error":"not found"}')

    def do_POST(self):
        length = int(self.headers.get("Content-Length", "0"))
        raw = self.rfile.read(length).decode("utf-8")
        try:
            req = json.loads(raw)
        except Exception:
            self._write(400, '{"error":"invalid json"}')
            return

        if self.path == "/api/tools":
            sid = req.get("server")
            if sid not in SERVERS:
                self._write(400, '{"error":"unknown server"}')
                return
            r = run_mcp(SERVERS[sid]["cmd"], [
                {"jsonrpc": "2.0", "method": "tools/list", "params": {}},
            ])
            resp = r["responses"][0] or {}
            tools = resp.get("result", {}).get("tools", []) if "result" in resp else []
            self._write(200, json.dumps({
                "init": r["init"],
                "tools": tools,
                "error": resp.get("error") if "error" in resp else None,
            }))
            return

        if self.path == "/api/call":
            sid = req.get("server")
            tool = req.get("tool")
            args = req.get("arguments", {}) or {}
            if sid not in SERVERS:
                self._write(400, '{"error":"unknown server"}')
                return
            r = run_mcp(SERVERS[sid]["cmd"], [
                {"jsonrpc": "2.0", "method": "tools/call",
                 "params": {"name": tool, "arguments": args}},
            ])
            self._write(200, json.dumps({"response": r["responses"][0]}))
            return

        self._write(404, '{"error":"not found"}')


def main():
    port = int(os.environ.get("PORT", "18090"))
    print(f"[mcp-ui] docker network={NET}", flush=True)
    print(f"[mcp-ui] listening on http://0.0.0.0:{port}", flush=True)
    http.server.ThreadingHTTPServer(("0.0.0.0", port), Handler).serve_forever()


if __name__ == "__main__":
    main()
