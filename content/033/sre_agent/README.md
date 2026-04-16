## sre_agent — local-LLM SRE analyst over MCP

A single-file agent (`sre_agent.py`) that a human on-call can actually use:

```
"Are there any firing alerts right now and what's the probable root cause?"
      │
      ▼
 ┌──────────────┐   tool_calls   ┌──────────────────────────┐
 │  Ollama      │ ─────────────► │  MCP: vm_query,           │
 │  (qwen2.5)   │ ◄───────────── │       prom_gs_get_alerts, │
 └──────────────┘    results     │       vm_query_range, …   │
        │                        └──────────────────────────┘
        ▼                                     │
  triage answer                 VictoriaMetrics / Prometheus
  with real numbers             (RED metrics from OTel spanmetrics)
```

- **Local model**, no external API — tested with `qwen2.5:14b` and `llama3.1:8b`
  on an RTX 4070 Ti Super (16 GB).
- **MCP tools** spawned as long-lived stdio subprocesses: the official
  [`VictoriaMetrics/mcp-victoriametrics`](https://github.com/VictoriaMetrics/mcp-victoriametrics)
  and [`giantswarm/mcp-prometheus`](https://github.com/giantswarm/mcp-prometheus)
  (the latter for Alertmanager).
- **RED signals from OTel spanmetrics**: reuses the `../otel-red` stack
  (Flask demo-app with scripted latency profiles + loadgen + OTel collector
  + Prometheus + Alertmanager + VictoriaMetrics + Grafana).

### Files

| File | What it is |
| --- | --- |
| `sre_agent.py`        | The agent: MCP stdio client, Ollama tool-calling loop, pretty-printer. |
| `smoke_mcp.py`        | Wire-check: spawns the MCPs and hits each tool once, no LLM. |
| `sre_questions.txt`   | The 5 canonical questions a human on-call would ask. |
| `results/*.json`      | Full transcripts: per-question tool calls + arguments + raw MCP results + final answer. |
| `results/*.log`       | Colored human-readable run log. |

### Run

The `otel-red` stack must already be up (that's where the RED signals come
from) — if it isn't: `cd ../otel-red && docker compose up -d --build` and
wait 90 s. Then:

```sh
# 1. Install Ollama and pull a model that does tool-calling
ollama pull qwen2.5:14b         # primary — strongest tool-use at 8 GB VRAM
ollama pull llama3.1:8b         # fallback — faster but weaker on multi-step

# 2. Smoke-test MCP wiring (no LLM, sub-second)
python3 smoke_mcp.py

# 3. One question, verbose trace
python3 sre_agent.py --model qwen2.5:14b \
  "Are there any firing alerts right now and what's the probable root cause?"

# 4. Full SRE playbook + saved transcript
python3 sre_agent.py --model qwen2.5:14b \
  --batch sre_questions.txt --transcript results/qwen2.5-14b.json
```

### Model comparison on the 5-question playbook

| Question | qwen2.5:14b | llama3.1:8b |
| --- | --- | --- |
| Q1. rate per endpoint                      | ✅ 1 call, correct | ✅ 1 call, correct |
| Q2. overall error rate                     | ✅ 4.55%            | ✅ 3.34%            |
| Q3. p95 per endpoint                       | ✅ /api/slow = 1743 ms | ✅ /api/slow = 1800 ms |
| Q4. slowest vs 300 ms SLO                  | ✅ +1442 ms         | ✅ +1500 ms         |
| Q5. firing alerts + probable root cause    | ✅ **3 tools in parallel** — correlated alert → slowest endpoint | ⚠️ lists alert only, describes next queries as *plan text* instead of calling them |
| total wall time for 5 questions            | 25.3 s              | 13.0 s              |
| total tool calls                           | 7                   | 5                   |

Takeaways:

- **llama3.1:8b is 2× faster** and fine for single-fact triage (Q1–Q4).
- **qwen2.5:14b is the one you actually want on-call** — it's the only one
  that did the multi-tool correlation on Q5 (alerts → error rate → p95 → point
  at the slowest endpoint), which is the whole reason you'd call an agent
  instead of running PromQL by hand.
- Both models handled PromQL syntax correctly once the system prompt gave
  them copy-ready queries and explicitly said *don't pass a `time` argument*.
  Without that guardrail, llama3.1 invented `time="now()"` and qwen tried to
  pass the PromQL *label* (`ERROR_RATIO_OVERALL`) as the query string.

### Why this structure is production-shaped

- **Separation of concerns.** The agent doesn't speak HTTP to any observability
  backend — it only speaks MCP. Swap VictoriaMetrics for Mimir, or Prometheus
  for Thanos, by changing the MCP command line. Nothing in the prompt or loop
  changes.
- **No secrets.** The MCP subprocess inherits env vars; credentials stay out
  of the model's context.
- **Reproducible runs.** `--transcript` dumps every tool call + argument + raw
  MCP result + final answer as JSON. Good for PR reviews, regression tests,
  and offline eval — diff two transcripts and you see exactly what the model
  asked differently.
- **Bounded blast radius.** The agent only gets *read-only* tools (query,
  labels, series, alerts). There is no tool that can silence an alert, change
  a rule, or write metrics. An SRE can run this against prod without needing
  an escape-hatch kill-switch.
- **Tool allowlist.** `VM_ALLOW` in `sre_agent.py` is the shortlist of vm-mcp
  tools worth exposing — skipping `documentation` (blows up context) and
  `metrics_metadata` (noisy). Add more only as use cases demand.

### Example transcript — Q5 on qwen2.5:14b

```
━━ Q ━━ Are there any firing alerts right now? …
[1] LLM → wants 3 tool call(s)
[2]   ↳ call prom_gs_get_alerts({"state": "firing"})
[3]   ← Active Alerts: DemoAppHighLatencyP95 firing, Value=790.47
[4]   ↳ call vm_query({"query": "sum(rate(...status_code=\"STATUS_CODE_ERROR\"…"})
[5]   ← 0.0297   (error ratio ≈ 3%)
[6]   ↳ call vm_query({"query": "histogram_quantile(0.95, sum by (le,span_name) …"})
[7]   ← /api/fast=23.89  /api/flaky=237.14  /api/medium=230.86  /api/slow=1775
[8] LLM → final answer

━━ A ━━
There is a firing alert for the `demo-app` service indicating that the p95
latency has exceeded 300ms. The overall error ratio is 2.98%, relatively
low but still concerning. The p95 latencies per endpoint are:
- GET /api/fast: 23.89 ms
- GET /api/flaky: 237.14 ms
- GET /api/medium: 230.86 ms
- GET /api/slow: 1775 ms

The `GET /api/slow` endpoint has the highest p95 latency, which is likely
the primary driver of the service-level p95 breach.

Next step: Investigate the backend or database queries associated with the
`/api/slow` endpoint to identify and address any performance bottlenecks.
```

All numbers above came from real tool calls against the running stack — no
hallucinated values. The fake demo-app is designed so the expected answers
are obvious (`/api/slow` has a scripted 400–1200 ms profile, `/api/flaky`
returns 500 on ~35 % of requests), which makes it easy to spot when the
model is making things up.

### What to change next for a real deployment

1. **Point at your real backends.** Replace the `NET` and `VM_INSTANCE_ENTRYPOINT`
   (and the Prometheus URL) in `spawn_mcps()` with your cluster's endpoints.
2. **Scope metrics to one service.** The `SYSTEM_PROMPT` hardcodes
   `service_name="demo-app"`. Parameterize it per user or per route so the
   agent can only read what the user is allowed to read.
3. **Add an Alertmanager silence tool (write-capable).** Keep it behind a
   confirmation gate — the agent proposes, a human approves.
4. **Wrap it in a Slack bot.** Each Slack message → one `run_agent()` call
   → post the final answer. Transcripts go to object storage keyed by thread.
5. **Cache expensive queries.** `vm_query_range` over long windows is
   re-issued every turn; wrap the binding with a short-TTL LRU.
