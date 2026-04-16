#!/usr/bin/env python3
"""Smoke-test the MCP wiring without invoking any LLM.

Spawns the same two MCP subprocesses sre_agent.py uses and calls each tool
once with a known-good argument set against the otel-red backend. If this
passes, the LLM harness can only fail for LLM reasons (tool schema mismatch,
bad reasoning, etc.), not wiring.
"""

import json
import sys
from sre_agent import spawn_mcps, build_bindings

SMOKE = [
    ("vm_query", {"query": 'sum by (span_name) (rate(traces_spanmetrics_calls_total{service_name="demo-app"}[1m]))'}),
    ("vm_labels", {}),
    ("vm_label_values", {"label_name": "span_name"}),
    ("prom_gs_get_alerts", {}),
]


def main() -> int:
    mcps = spawn_mcps()
    try:
        bindings = build_bindings(mcps)
        by_name = {b.public_name: b for b in bindings}
        print(f"exposed {len(bindings)} tools")
        print("  " + ", ".join(sorted(by_name)))
        print()
        for name, args in SMOKE:
            b = by_name.get(name)
            if not b:
                print(f"SKIP {name}: not exposed")
                continue
            res = b.impl(args)
            preview = json.dumps(res, ensure_ascii=False)
            if len(preview) > 300:
                preview = preview[:300] + " …"
            print(f"[{name}] {preview}")
    finally:
        for c in mcps.values():
            c.stop()
    return 0


if __name__ == "__main__":
    sys.exit(main())
