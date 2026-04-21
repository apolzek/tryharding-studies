#!/usr/bin/env python3
"""
13 — Latência de open(): histograma do tempo entre sys_enter_openat e sys_exit_openat.

Ensina a casar "entry" e "exit" usando BPF_HASH por TID.

Run: sudo python3 13_file_open_latency.py   (Ctrl-C para histograma)
Test: sudo make test-13
"""
from bcc import BPF

prog = r"""
BPF_HASH(start, u32, u64);
BPF_HISTOGRAM(dist);

TRACEPOINT_PROBE(syscalls, sys_enter_openat) {
    u32 tid = bpf_get_current_pid_tgid();
    u64 ts = bpf_ktime_get_ns();
    start.update(&tid, &ts);
    return 0;
}

TRACEPOINT_PROBE(syscalls, sys_exit_openat) {
    u32 tid = bpf_get_current_pid_tgid();
    u64 *ts = start.lookup(&tid);
    if (!ts) return 0;
    u64 delta = bpf_ktime_get_ns() - *ts;
    start.delete(&tid);
    dist.increment(bpf_log2l(delta / 1000));   /* microseconds */
    return 0;
}
"""

b = BPF(text=prog)
print("measuring openat() latency... Ctrl-C")
try:
    while True: pass
except KeyboardInterrupt: pass

print("\nopenat() latency (us):")
b["dist"].print_log2_hist("us")
