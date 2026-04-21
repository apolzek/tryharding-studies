#!/usr/bin/env python3
"""
08 — connect(2) snoop (IPv4): quem conecta para onde.

Usa kprobe em tcp_v4_connect para pegar sock + porta.

Run:     sudo python3 08_tcp_connect.py
Test: sudo make test-08
Trigger: curl http://example.com
"""
from bcc import BPF
import socket, struct

prog = r"""
#include <net/sock.h>
struct data_t { u32 pid; u32 saddr; u32 daddr; u16 dport; char comm[16]; };
BPF_PERF_OUTPUT(events);

int kp_connect(struct pt_regs *ctx, struct sock *sk) {
    struct data_t d = {};
    d.pid   = bpf_get_current_pid_tgid() >> 32;
    d.saddr = sk->__sk_common.skc_rcv_saddr;
    d.daddr = sk->__sk_common.skc_daddr;
    d.dport = bpf_ntohs(sk->__sk_common.skc_dport);
    bpf_get_current_comm(&d.comm, sizeof(d.comm));
    events.perf_submit(ctx, &d, sizeof(d));
    return 0;
}
"""

b = BPF(text=prog)
b.attach_kprobe(event="tcp_v4_connect", fn_name="kp_connect")

print(f"{'PID':<8}{'COMM':<16}{'SRC':<16}{'DST':<16}{'DPORT'}")
def cb(cpu, data, size):
    e = b["events"].event(data)
    s = socket.inet_ntoa(struct.pack("I", e.saddr))
    d = socket.inet_ntoa(struct.pack("I", e.daddr))
    print(f"{e.pid:<8}{e.comm.decode():<16}{s:<16}{d:<16}{e.dport}")
b["events"].open_perf_buffer(cb)
try:
    while True: b.perf_buffer_poll()
except KeyboardInterrupt: pass
