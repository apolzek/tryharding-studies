#!/usr/bin/env python3
"""
11 — unlinkat(2) snoop: detecta arquivos sendo deletados.

Run: sudo python3 11_unlink_snoop.py    (em outro terminal: rm /tmp/x)
Test: sudo make test-11
"""
from bcc import BPF

prog = r"""
struct data_t { u32 pid; char comm[16]; char fname[128]; };
BPF_PERF_OUTPUT(events);

TRACEPOINT_PROBE(syscalls, sys_enter_unlinkat) {
    struct data_t d = {};
    d.pid = bpf_get_current_pid_tgid() >> 32;
    bpf_get_current_comm(&d.comm, sizeof(d.comm));
    bpf_probe_read_user_str(&d.fname, sizeof(d.fname), (void *)args->pathname);
    events.perf_submit(args, &d, sizeof(d));
    return 0;
}
"""

b = BPF(text=prog)
print(f"{'PID':<8}{'COMM':<16}FILE")
def cb(cpu, data, size):
    e = b["events"].event(data)
    print(f"{e.pid:<8}{e.comm.decode():<16}{e.fname.decode('utf-8','replace')}")
b["events"].open_perf_buffer(cb)
try:
    while True: b.perf_buffer_poll()
except KeyboardInterrupt: pass
