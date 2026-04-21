#!/usr/bin/env python3
"""
10 — kill(2) snoop: quem mandou signal para quem.

Útil para descobrir quem está matando um daemon misteriosamente.

Run: sudo python3 10_kill_snoop.py
Test: sudo make test-10
"""
from bcc import BPF

prog = r"""
struct data_t { u32 pid; s32 tpid; s32 sig; char comm[16]; };
BPF_PERF_OUTPUT(events);

TRACEPOINT_PROBE(syscalls, sys_enter_kill) {
    struct data_t d = {};
    d.pid  = bpf_get_current_pid_tgid() >> 32;
    d.tpid = args->pid;
    d.sig  = args->sig;
    bpf_get_current_comm(&d.comm, sizeof(d.comm));
    events.perf_submit(args, &d, sizeof(d));
    return 0;
}
"""

b = BPF(text=prog)
print(f"{'PID':<8}{'COMM':<16}{'-> TPID':<10}SIG")
def cb(cpu, data, size):
    e = b["events"].event(data)
    print(f"{e.pid:<8}{e.comm.decode():<16}{e.tpid:<10}{e.sig}")
b["events"].open_perf_buffer(cb)
try:
    while True: b.perf_buffer_poll()
except KeyboardInterrupt: pass
