#!/usr/bin/env python3
"""
18 — Detecta quem chama getdents64 (listagem de diretório).

Útil para ver scripts fazendo varreduras em /proc, /sys etc.

Run: sudo python3 18_dir_listings.py   (trigger: ls /proc)
Test: sudo make test-18
"""
from bcc import BPF

prog = r"""
struct data_t { u32 pid; char comm[16]; int fd; };
BPF_PERF_OUTPUT(events);

TRACEPOINT_PROBE(syscalls, sys_enter_getdents64) {
    struct data_t d = {};
    d.pid = bpf_get_current_pid_tgid() >> 32;
    d.fd  = args->fd;
    bpf_get_current_comm(&d.comm, sizeof(d.comm));
    events.perf_submit(args, &d, sizeof(d));
    return 0;
}
"""

b = BPF(text=prog)
print(f"{'PID':<8}{'COMM':<16}{'FD':>6}")
def cb(cpu, data, size):
    e = b["events"].event(data)
    print(f"{e.pid:<8}{e.comm.decode():<16}{e.fd:>6}")
b["events"].open_perf_buffer(cb)
try:
    while True: b.perf_buffer_poll()
except KeyboardInterrupt: pass
