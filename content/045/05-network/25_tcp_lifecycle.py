#!/usr/bin/env python3
"""
25 — Ciclo de vida TCP: captura mudanças de estado via tcp_set_state.

Útil para entender CLOSE_WAIT, TIME_WAIT, retransmit bursts etc.

Run: sudo python3 25_tcp_lifecycle.py
Test: sudo make test-25
"""
from bcc import BPF
import socket, struct

STATES = {
    1: "ESTABLISHED", 2: "SYN_SENT", 3: "SYN_RECV", 4: "FIN_WAIT1",
    5: "FIN_WAIT2",   6: "TIME_WAIT", 7: "CLOSE",   8: "CLOSE_WAIT",
    9: "LAST_ACK",    10: "LISTEN",   11: "CLOSING",
}

prog = r"""
#include <net/sock.h>
struct data_t { u32 pid; u32 saddr; u32 daddr; u16 sport; u16 dport;
                int oldstate; int newstate; char comm[16]; };
BPF_PERF_OUTPUT(events);

int kp_set_state(struct pt_regs *ctx, struct sock *sk, int newstate) {
    struct data_t d = {};
    d.pid   = bpf_get_current_pid_tgid() >> 32;
    d.oldstate = sk->__sk_common.skc_state;
    d.newstate = newstate;
    d.saddr = sk->__sk_common.skc_rcv_saddr;
    d.daddr = sk->__sk_common.skc_daddr;
    d.sport = sk->__sk_common.skc_num;
    d.dport = bpf_ntohs(sk->__sk_common.skc_dport);
    bpf_get_current_comm(&d.comm, sizeof(d.comm));
    events.perf_submit(ctx, &d, sizeof(d));
    return 0;
}
"""

b = BPF(text=prog)
b.attach_kprobe(event="tcp_set_state", fn_name="kp_set_state")

print(f"{'PID':<8}{'COMM':<16}{'SRC':<22}{'DST':<22}OLD->NEW")
def cb(cpu, data, size):
    e = b["events"].event(data)
    s = f"{socket.inet_ntoa(struct.pack('I', e.saddr))}:{e.sport}"
    d = f"{socket.inet_ntoa(struct.pack('I', e.daddr))}:{e.dport}"
    print(f"{e.pid:<8}{e.comm.decode():<16}{s:<22}{d:<22}"
          f"{STATES.get(e.oldstate,e.oldstate)}->{STATES.get(e.newstate,e.newstate)}")
b["events"].open_perf_buffer(cb)
try:
    while True: b.perf_buffer_poll()
except KeyboardInterrupt: pass
