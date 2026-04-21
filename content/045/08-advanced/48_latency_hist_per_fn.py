#!/usr/bin/env python3
"""
48 — Histograma genérico de latência para qualquer função do kernel.

Uso: sudo python3 48_latency_hist_per_fn.py vfs_read
     sudo python3 48_latency_hist_per_fn.py tcp_sendmsg
"""
import sys
from bcc import BPF

if len(sys.argv) < 2:
    sys.exit("usage: 48_latency_hist_per_fn.py <kernel_fn>")
FN = sys.argv[1]

prog = r"""
BPF_HASH(start, u32, u64);
BPF_HISTOGRAM(dist);

int kp(struct pt_regs *ctx) {
    u32 tid = bpf_get_current_pid_tgid();
    u64 t = bpf_ktime_get_ns();
    start.update(&tid, &t);
    return 0;
}
int krp(struct pt_regs *ctx) {
    u32 tid = bpf_get_current_pid_tgid();
    u64 *t = start.lookup(&tid);
    if (!t) return 0;
    dist.increment(bpf_log2l((bpf_ktime_get_ns() - *t) / 1000));
    start.delete(&tid);
    return 0;
}
"""

b = BPF(text=prog)
b.attach_kprobe(event=FN, fn_name="kp")
b.attach_kretprobe(event=FN, fn_name="krp")
print(f"measuring {FN}() latency. Ctrl-C")
try:
    while True: pass
except KeyboardInterrupt: pass
print(f"\n{FN}() latency (us):")
b["dist"].print_log2_hist("us")
