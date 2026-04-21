#!/usr/bin/env python3
"""
09 — accept(2) snoop: cada nova conexão TCP aceita localmente.

Mostra kretprobe (pega o valor de retorno de inet_csk_accept).

Run: sudo python3 09_accept_snoop.py   (depois: nc -l 9000 &  curl localhost:9000)
Test: sudo make test-09
"""
from bcc import BPF
import socket, struct

prog = r"""
#include <net/sock.h>
struct data_t { u32 pid; u32 saddr; u32 daddr; u16 lport; char comm[16]; };
BPF_PERF_OUTPUT(events);

int krp_accept(struct pt_regs *ctx) {
    struct sock *sk = (struct sock *)PT_REGS_RC(ctx);
    if (!sk) return 0;
    struct data_t d = {};
    d.pid   = bpf_get_current_pid_tgid() >> 32;
    d.saddr = sk->__sk_common.skc_rcv_saddr;
    d.daddr = sk->__sk_common.skc_daddr;
    d.lport = sk->__sk_common.skc_num;
    bpf_get_current_comm(&d.comm, sizeof(d.comm));
    events.perf_submit(ctx, &d, sizeof(d));
    return 0;
}
"""

b = BPF(text=prog)
b.attach_kretprobe(event="inet_csk_accept", fn_name="krp_accept")

print(f"{'PID':<8}{'COMM':<16}{'SRC':<16}{'DST':<16}{'LPORT'}")
def cb(cpu, data, size):
    e = b["events"].event(data)
    s = socket.inet_ntoa(struct.pack("I", e.saddr))
    d = socket.inet_ntoa(struct.pack("I", e.daddr))
    print(f"{e.pid:<8}{e.comm.decode():<16}{s:<16}{d:<16}{e.lport}")
b["events"].open_perf_buffer(cb)
try:
    while True: b.perf_buffer_poll()
except KeyboardInterrupt: pass
