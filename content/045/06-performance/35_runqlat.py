#!/usr/bin/env python3
"""
35 — Runqueue latency: tempo entre wakeup e on-CPU (scheduler latency).

Run: sudo python3 35_runqlat.py   (Ctrl-C para histograma)
Test: sudo make test-35
"""
from bcc import BPF

prog = r"""
BPF_HASH(start, u32, u64);
BPF_HISTOGRAM(dist);

TRACEPOINT_PROBE(sched, sched_wakeup) {
    u64 ts = bpf_ktime_get_ns();
    u32 pid = args->pid;
    start.update(&pid, &ts);
    return 0;
}
TRACEPOINT_PROBE(sched, sched_wakeup_new) {
    u64 ts = bpf_ktime_get_ns();
    u32 pid = args->pid;
    start.update(&pid, &ts);
    return 0;
}
TRACEPOINT_PROBE(sched, sched_switch) {
    u32 pid = args->next_pid;
    u64 *ts = start.lookup(&pid);
    if (!ts) return 0;
    u64 dt = (bpf_ktime_get_ns() - *ts) / 1000;
    start.delete(&pid);
    dist.increment(bpf_log2l(dt));
    return 0;
}
"""

b = BPF(text=prog)
print("measuring runqueue latency. Ctrl-C")
try:
    while True: pass
except KeyboardInterrupt: pass

print("\nrunqueue latency (us):")
b["dist"].print_log2_hist("us")
