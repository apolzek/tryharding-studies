#!/usr/bin/env python3
"""
26 — Retransmissões TCP em tempo real (sinaliza perda de pacote).

Kprobe em tcp_retransmit_skb.

Run: sudo python3 26_tcp_retransmit.py
Test: sudo make test-26
"""
from bcc import BPF
import socket, struct

prog = r"""
#include <net/sock.h>
struct data_t { u32 saddr; u32 daddr; u16 dport; u16 sport; };
BPF_PERF_OUTPUT(events);

int kp(struct pt_regs *ctx, struct sock *sk) {
    struct data_t d = {};
    d.saddr = sk->__sk_common.skc_rcv_saddr;
    d.daddr = sk->__sk_common.skc_daddr;
    d.sport = sk->__sk_common.skc_num;
    d.dport = bpf_ntohs(sk->__sk_common.skc_dport);
    events.perf_submit(ctx, &d, sizeof(d));
    return 0;
}
"""

b = BPF(text=prog)
b.attach_kprobe(event="tcp_retransmit_skb", fn_name="kp")

print(f"{'SRC':<22}{'DST':<22}")
def cb(cpu, data, size):
    e = b["events"].event(data)
    s = f"{socket.inet_ntoa(struct.pack('I', e.saddr))}:{e.sport}"
    d = f"{socket.inet_ntoa(struct.pack('I', e.daddr))}:{e.dport}"
    print(f"{s:<22}{d:<22}")
b["events"].open_perf_buffer(cb)
try:
    while True: b.perf_buffer_poll()
except KeyboardInterrupt: pass
