#!/usr/bin/env python3
"""
28 — UDP send snoop: todos os datagramas UDP enviados.

Run: sudo python3 28_udp_snoop.py
Test: sudo make test-28
"""
from bcc import BPF
import socket, struct

prog = r"""
#include <net/sock.h>
struct data_t { u32 pid; u32 saddr; u32 daddr; u16 dport; u16 sport;
                u32 len; char comm[16]; };
BPF_PERF_OUTPUT(events);

int kp_send(struct pt_regs *ctx, struct sock *sk, struct msghdr *msg, size_t len) {
    struct data_t d = {};
    d.pid   = bpf_get_current_pid_tgid() >> 32;
    d.saddr = sk->__sk_common.skc_rcv_saddr;
    d.daddr = sk->__sk_common.skc_daddr;
    d.sport = sk->__sk_common.skc_num;
    d.dport = bpf_ntohs(sk->__sk_common.skc_dport);
    d.len   = len;
    bpf_get_current_comm(&d.comm, sizeof(d.comm));
    events.perf_submit(ctx, &d, sizeof(d));
    return 0;
}
"""

b = BPF(text=prog)
b.attach_kprobe(event="udp_sendmsg", fn_name="kp_send")

print(f"{'PID':<8}{'COMM':<16}{'SRC':<22}{'DST':<22}{'LEN':>6}")
def cb(cpu, data, size):
    e = b["events"].event(data)
    s = f"{socket.inet_ntoa(struct.pack('I', e.saddr))}:{e.sport}"
    d = f"{socket.inet_ntoa(struct.pack('I', e.daddr))}:{e.dport}"
    print(f"{e.pid:<8}{e.comm.decode():<16}{s:<22}{d:<22}{e.len:>6}")
b["events"].open_perf_buffer(cb)
try:
    while True: b.perf_buffer_poll()
except KeyboardInterrupt: pass
