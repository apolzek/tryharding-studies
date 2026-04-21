#!/usr/bin/env python3
"""
05 — BPF_HISTOGRAM: histograma log2 de tamanhos de write().

Ilustra o helper `bpf_log2l` e o print pronto `b[...].print_log2_hist`.

Run: sudo python3 05_histogram.py   (Ctrl-C para imprimir o histograma)
Test: sudo make test-05
"""
from bcc import BPF

prog = r"""
BPF_HISTOGRAM(dist);

int on_write(struct pt_regs *ctx, unsigned int fd, const char *buf, size_t count) {
    dist.increment(bpf_log2l(count));
    return 0;
}
"""

b = BPF(text=prog)
b.attach_kprobe(event=b.get_syscall_fnname("write"), fn_name="on_write")

print("counting write() sizes (log2). Ctrl-C to print.")
try:
    while True:
        pass
except KeyboardInterrupt:
    pass

print("\nwrite() size (bytes):")
b["dist"].print_log2_hist("bytes")
