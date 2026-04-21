#!/usr/bin/env python3
"""
27 — Histograma de RTT medido por tcp_rcv_established (srtt_us).

Mostra distribuição de RTT dos sockets TCP estabelecidos.

Run: sudo python3 27_tcp_rtt_hist.py   (Ctrl-C para histograma)
Test: sudo make test-27
"""
from bcc import BPF

prog = r"""
#include <net/sock.h>
#include <net/inet_sock.h>
#include <net/tcp.h>
BPF_HISTOGRAM(dist);

int kp(struct pt_regs *ctx, struct sock *sk) {
    struct tcp_sock *ts = (struct tcp_sock *)sk;
    u32 srtt_us = ts->srtt_us >> 3;  /* stored as 8x scaled */
    if (!srtt_us) return 0;
    dist.increment(bpf_log2l(srtt_us));
    return 0;
}
"""

b = BPF(text=prog)
b.attach_kprobe(event="tcp_rcv_established", fn_name="kp")

print("sampling TCP srtt... Ctrl-C")
try:
    while True: pass
except KeyboardInterrupt: pass

print("\nTCP srtt (us):")
b["dist"].print_log2_hist("us")
