#!/usr/bin/env python3
"""
33 — Biolatency: histograma de latência de I/O de bloco (disco).

Anexa em block_rq_issue / block_rq_complete via tracepoints.

Run: sudo python3 33_biolatency.py   (Ctrl-C para histograma)
Test: sudo make test-33
"""
from bcc import BPF

prog = r"""
BPF_HASH(start, u64, u64);
BPF_HISTOGRAM(dist);

TRACEPOINT_PROBE(block, block_rq_issue) {
    u64 rq = (u64)args->dev;   /* usa dev+sector como key composto */
    u64 ts = bpf_ktime_get_ns();
    start.update(&rq, &ts);
    return 0;
}
TRACEPOINT_PROBE(block, block_rq_complete) {
    u64 rq = (u64)args->dev;
    u64 *ts = start.lookup(&rq);
    if (!ts) return 0;
    u64 dt = (bpf_ktime_get_ns() - *ts) / 1000;
    start.delete(&rq);
    dist.increment(bpf_log2l(dt));
    return 0;
}
"""

b = BPF(text=prog)
print("measuring block I/O latency. Ctrl-C to plot.")
try:
    while True: pass
except KeyboardInterrupt: pass

print("\nblock I/O latency (us):")
b["dist"].print_log2_hist("us")
