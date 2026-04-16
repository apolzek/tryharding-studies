#!/usr/bin/env python3
"""
sre_agent — local-LLM SRE analyst that talks to Prometheus/VictoriaMetrics
through MCP tools.

Pipeline:

    user question → Ollama (tool-calling) → MCP tools → Ollama → final answer

Two MCP servers are spawned as long-lived stdio subprocesses and their tools
are exposed to the model via Ollama's function-calling interface:

  - vm_mcp        → VictoriaMetrics/mcp-victoriametrics   (query, query_range,
                    labels, label_values, series, tsdb_status, alerts, ...)
  - prom_mcp_gs   → giantswarm/prometheus-mcp-server      (get_alerts)

Everything points at the otel-red stack (demo-app + OTel collector + Prom + VM
+ Alertmanager) that lives on the `mcp033-otel-red_obs` Docker network. The
demo-app is instrumented with OTel zero-code auto-instrumentation and a
spanmetrics connector, so the backend exposes the canonical RED surface:

  traces_spanmetrics_calls_total{service_name,span_name,status_code,...}
  traces_spanmetrics_duration_milliseconds_bucket{le, service_name, span_name}

CLI:
    python3 sre_agent.py "what is the p95 latency per endpoint right now?"
    python3 sre_agent.py --model llama3.1:8b "..."
    python3 sre_agent.py --transcript results/t.json "..."
"""

from __future__ import annotations

import argparse
import json
import os
import queue
import subprocess
import sys
import threading
import time
from dataclasses import dataclass, field
from typing import Any, Callable

import urllib.request


# --------------------------------------------------------------------------- #
# MCP stdio client                                                            #
# --------------------------------------------------------------------------- #


class MCPClient:
    """Minimal JSON-RPC 2.0 MCP client over a long-lived stdio subprocess."""

    def __init__(self, name: str, cmd: list[str]):
        self.name = name
        self.cmd = cmd
        self.proc: subprocess.Popen | None = None
        self._q: queue.Queue[str | None] = queue.Queue()
        self._id = 0
        self._lock = threading.Lock()
        self.tools: list[dict] = []

    def start(self) -> None:
        self.proc = subprocess.Popen(
            self.cmd,
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.DEVNULL,
            bufsize=0,
        )
        threading.Thread(target=self._reader, daemon=True).start()
        self._rpc("initialize", {
            "protocolVersion": "2025-03-26",
            "capabilities": {},
            "clientInfo": {"name": "sre_agent", "version": "0.1.0"},
        })
        self._notify("notifications/initialized", {})
        res = self._rpc("tools/list", {})
        self.tools = (res or {}).get("tools", []) if res else []

    def stop(self) -> None:
        if self.proc and self.proc.poll() is None:
            try:
                self.proc.stdin.close()  # type: ignore[union-attr]
            except Exception:
                pass
            try:
                self.proc.wait(timeout=2)
            except Exception:
                self.proc.kill()

    def call(self, tool_name: str, args: dict, timeout: float = 30.0) -> dict:
        res = self._rpc("tools/call", {"name": tool_name, "arguments": args}, timeout=timeout)
        if res is None:
            return {"error": "mcp timeout or closed"}
        content = res.get("content") if isinstance(res, dict) else None
        if isinstance(content, list) and content and isinstance(content[0], dict):
            text = content[0].get("text")
            if isinstance(text, str):
                try:
                    return {"result": json.loads(text)}
                except Exception:
                    return {"result": text}
        return {"result": res}

    def _reader(self):
        assert self.proc and self.proc.stdout
        for line in iter(self.proc.stdout.readline, b""):
            self._q.put(line.decode("utf-8", errors="replace"))
        self._q.put(None)

    def _send(self, obj: dict) -> None:
        assert self.proc and self.proc.stdin
        data = (json.dumps(obj) + "\n").encode("utf-8")
        self.proc.stdin.write(data)
        self.proc.stdin.flush()

    def _notify(self, method: str, params: dict) -> None:
        self._send({"jsonrpc": "2.0", "method": method, "params": params})

    def _rpc(self, method: str, params: dict, timeout: float = 15.0) -> dict | None:
        with self._lock:
            self._id += 1
            rid = self._id
            self._send({"jsonrpc": "2.0", "id": rid, "method": method, "params": params})
            deadline = time.time() + timeout
            while time.time() < deadline:
                try:
                    line = self._q.get(timeout=0.5)
                except queue.Empty:
                    if self.proc and self.proc.poll() is not None:
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
                if msg.get("id") == rid:
                    if "error" in msg:
                        return {"__error__": msg["error"]}
                    return msg.get("result")
            return None


# --------------------------------------------------------------------------- #
# MCP tool → Ollama function schema translation                               #
# --------------------------------------------------------------------------- #


# Some tools take heavy/rarely-needed args or can explode context (e.g. full
# documentation dumps). Only expose tools that are useful for SRE triage.
VM_ALLOW = {
    "query",
    "query_range",
    "labels",
    "label_values",
    "series",
    "tsdb_status",
    "alerts",
    "active_queries",
    "top_queries",
}


@dataclass
class ToolBinding:
    """One callable tool as seen by the LLM."""
    public_name: str                   # what the model sees
    description: str
    schema: dict                       # JSON schema for parameters
    impl: Callable[[dict], dict]       # executes the tool


def build_bindings(mcps: dict[str, MCPClient]) -> list[ToolBinding]:
    bindings: list[ToolBinding] = []
    for prefix, client in mcps.items():
        for t in client.tools:
            name = t.get("name")
            if prefix == "vm" and name not in VM_ALLOW:
                continue
            if prefix == "prom_gs" and name != "get_alerts":
                continue
            public_name = f"{prefix}_{name}"
            schema = t.get("inputSchema") or {"type": "object", "properties": {}}
            desc = t.get("description") or ""
            desc = desc.strip().split("\n\n")[0][:500]
            bindings.append(
                ToolBinding(
                    public_name=public_name,
                    description=f"[{prefix}] {desc}",
                    schema=schema,
                    impl=_mk_impl(client, name),
                )
            )
    return bindings


def _mk_impl(client: MCPClient, tool_name: str):
    def impl(args: dict) -> dict:
        return client.call(tool_name, args or {})
    return impl


def to_ollama_tools(bindings: list[ToolBinding]) -> list[dict]:
    return [
        {
            "type": "function",
            "function": {
                "name": b.public_name,
                "description": b.description,
                "parameters": b.schema,
            },
        }
        for b in bindings
    ]


# --------------------------------------------------------------------------- #
# Ollama chat with tool-calling loop                                          #
# --------------------------------------------------------------------------- #


OLLAMA_URL = os.environ.get("OLLAMA_URL", "http://localhost:11434")


def ollama_chat(model: str, messages: list[dict], tools: list[dict]) -> dict:
    body = {
        "model": model,
        "messages": messages,
        "tools": tools,
        "stream": False,
        "options": {"temperature": 0.1, "num_ctx": 8192},
    }
    req = urllib.request.Request(
        f"{OLLAMA_URL}/api/chat",
        data=json.dumps(body).encode("utf-8"),
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    with urllib.request.urlopen(req, timeout=300) as r:
        return json.loads(r.read().decode("utf-8"))


SYSTEM_PROMPT = """You are an SRE on-call engineer analysing the service `demo-app`.

You have MCP tools that query VictoriaMetrics (RED metrics from OpenTelemetry
spanmetrics) and Prometheus Alertmanager. You MUST call tools to get real
numbers before answering. Never invent values.

Metric names (spanmetrics output):
  traces_spanmetrics_calls_total{service_name, span_name, status_code}
  traces_spanmetrics_duration_milliseconds_bucket{le, service_name, span_name}

Example PromQL queries (pass the PromQL expression itself as `query`, not the label):

Request rate per endpoint:
`sum by (span_name) (rate(traces_spanmetrics_calls_total{service_name="demo-app"}[1m]))`

Overall error ratio (0.0 = no errors, 1.0 = all errors):
`sum(rate(traces_spanmetrics_calls_total{service_name="demo-app",status_code="STATUS_CODE_ERROR"}[1m])) / sum(rate(traces_spanmetrics_calls_total{service_name="demo-app"}[1m]))`

Error rate per endpoint:
`sum by (span_name) (rate(traces_spanmetrics_calls_total{service_name="demo-app",status_code="STATUS_CODE_ERROR"}[1m]))`

p95 latency (ms) per endpoint:
`histogram_quantile(0.95, sum by (le, span_name) (rate(traces_spanmetrics_duration_milliseconds_bucket{service_name="demo-app"}[1m])))`

Tool usage rules:
 - Call `vm_query` with ONLY the `query` argument. Do NOT pass a `time` argument.
 - If a query returns an error, re-read the query carefully (parenthesis count!) and call vm_query again with a fixed version. Do not give up after one failure.
 - Call `prom_gs_get_alerts` with empty arguments `{}` to list firing alerts.
 - Before the FINAL answer, ensure each claim is backed by a real tool result.

Final answer format:
 - 3-6 lines, plain text, no markdown tables.
 - Lead with the headline (what is broken / how bad).
 - Cite concrete numbers pulled from tool results.
 - Suggest ONE concrete next step an on-call would take.
"""


@dataclass
class TraceStep:
    kind: str                          # 'llm', 'tool_call', 'tool_result'
    data: Any


@dataclass
class AgentResult:
    question: str
    model: str
    final: str
    steps: list[TraceStep] = field(default_factory=list)
    elapsed_s: float = 0.0


def run_agent(
    model: str,
    question: str,
    bindings: list[ToolBinding],
    max_iters: int = 8,
) -> AgentResult:
    by_name = {b.public_name: b for b in bindings}
    tools = to_ollama_tools(bindings)
    messages: list[dict] = [
        {"role": "system", "content": SYSTEM_PROMPT},
        {"role": "user", "content": question},
    ]
    result = AgentResult(question=question, model=model, final="")
    t0 = time.time()

    for _ in range(max_iters):
        resp = ollama_chat(model, messages, tools)
        msg = resp.get("message") or {}
        tool_calls = msg.get("tool_calls") or []
        content = msg.get("content") or ""
        result.steps.append(TraceStep(kind="llm", data={"content": content, "tool_calls": tool_calls}))

        # If the model produced no tool calls, we're done.
        if not tool_calls:
            result.final = content.strip()
            break

        # Append the assistant message (with its tool_calls) and resolve each.
        messages.append({"role": "assistant", "content": content, "tool_calls": tool_calls})

        for tc in tool_calls:
            fn = (tc.get("function") or {})
            name = fn.get("name")
            raw_args = fn.get("arguments") or {}
            if isinstance(raw_args, str):
                try:
                    raw_args = json.loads(raw_args)
                except Exception:
                    raw_args = {}
            result.steps.append(TraceStep(kind="tool_call", data={"name": name, "arguments": raw_args}))

            binding = by_name.get(name)
            if binding is None:
                tool_result = {"error": f"unknown tool {name}"}
            else:
                try:
                    tool_result = binding.impl(raw_args)
                except Exception as e:
                    tool_result = {"error": f"{type(e).__name__}: {e}"}

            result.steps.append(TraceStep(kind="tool_result", data={"name": name, "result": tool_result}))
            messages.append({
                "role": "tool",
                "content": json.dumps(tool_result)[:12000],
            })

    result.elapsed_s = time.time() - t0
    return result


# --------------------------------------------------------------------------- #
# Pretty printer                                                              #
# --------------------------------------------------------------------------- #


C_RESET = "\033[0m"
C_DIM = "\033[2m"
C_BOLD = "\033[1m"
C_BLUE = "\033[34m"
C_GREEN = "\033[32m"
C_YELLOW = "\033[33m"
C_MAGENTA = "\033[35m"
C_CYAN = "\033[36m"


def _short(s: str, n: int = 400) -> str:
    s = s.strip()
    return s if len(s) <= n else s[:n] + " …"


def print_trace(r: AgentResult) -> None:
    print(f"\n{C_BOLD}━━ Q ━━{C_RESET} {r.question}")
    print(f"{C_DIM}model={r.model} elapsed={r.elapsed_s:.1f}s{C_RESET}\n")
    for i, st in enumerate(r.steps, 1):
        if st.kind == "llm":
            content = (st.data.get("content") or "").strip()
            tc = st.data.get("tool_calls") or []
            if tc:
                print(f"{C_BLUE}[{i}] LLM → wants {len(tc)} tool call(s){C_RESET}")
                if content:
                    print(f"    {C_DIM}thought:{C_RESET} {_short(content, 200)}")
            else:
                print(f"{C_GREEN}[{i}] LLM → final answer{C_RESET}")
        elif st.kind == "tool_call":
            name = st.data.get("name")
            args = json.dumps(st.data.get("arguments") or {}, ensure_ascii=False)
            print(f"{C_YELLOW}[{i}]   ↳ call {name}({_short(args, 300)}){C_RESET}")
        elif st.kind == "tool_result":
            name = st.data.get("name")
            res = st.data.get("result")
            preview = _short(json.dumps(res, ensure_ascii=False), 400)
            print(f"{C_MAGENTA}[{i}]   ← {name} → {preview}{C_RESET}")
    print(f"\n{C_CYAN}{C_BOLD}━━ A ━━{C_RESET}")
    print(r.final or "(no final answer)")
    print()


# --------------------------------------------------------------------------- #
# Entrypoint                                                                  #
# --------------------------------------------------------------------------- #


NET = "mcp033-otel-red_obs"


def spawn_mcps() -> dict[str, MCPClient]:
    vm_cmd = [
        "docker", "run", "--rm", "-i",
        "--network", NET,
        "-e", "VM_INSTANCE_ENTRYPOINT=http://victoriametrics:8428",
        "-e", "VM_INSTANCE_TYPE=single",
        "ghcr.io/victoriametrics/mcp-victoriametrics:latest",
        "--mode=stdio",
    ]
    prom_gs_cmd = [
        "docker", "run", "--rm", "-i",
        "--network", NET,
        "-e", "PROMETHEUS_URL=http://prometheus:9090",
        "local/mcp-prometheus:latest",
        "serve", "--transport=stdio",
    ]
    mcps = {
        "vm": MCPClient("vm", vm_cmd),
        "prom_gs": MCPClient("prom_gs", prom_gs_cmd),
    }
    for c in mcps.values():
        c.start()
    return mcps


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("question", nargs="?", help="single question to analyse")
    ap.add_argument("--model", default=os.environ.get("MODEL", "qwen2.5:14b"))
    ap.add_argument("--batch", help="file with one question per line")
    ap.add_argument("--transcript", help="save JSON transcript to file")
    args = ap.parse_args()

    if not args.question and not args.batch:
        ap.error("provide a question or --batch FILE")

    mcps = spawn_mcps()
    try:
        bindings = build_bindings(mcps)
        print(f"{C_DIM}mcp tools exposed to LLM: "
              f"{', '.join(b.public_name for b in bindings)}{C_RESET}\n")

        questions: list[str] = []
        if args.question:
            questions.append(args.question)
        if args.batch:
            with open(args.batch) as fh:
                questions.extend([ln.strip() for ln in fh if ln.strip() and not ln.startswith("#")])

        transcripts: list[dict] = []
        for q in questions:
            r = run_agent(args.model, q, bindings)
            print_trace(r)
            transcripts.append({
                "question": r.question,
                "model": r.model,
                "elapsed_s": r.elapsed_s,
                "final": r.final,
                "steps": [{"kind": s.kind, "data": s.data} for s in r.steps],
            })

        if args.transcript:
            with open(args.transcript, "w") as fh:
                json.dump(transcripts, fh, indent=2)
            print(f"{C_DIM}transcript → {args.transcript}{C_RESET}")

    finally:
        for c in mcps.values():
            c.stop()

    return 0


if __name__ == "__main__":
    sys.exit(main())
