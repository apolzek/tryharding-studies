#!/usr/bin/env python3
"""
07 — openat snoop: arquivo por arquivo que o sistema abre.

Bom exemplo de `bpf_probe_read_user_str` para copiar string de userspace.

Run: sudo python3 07_openat_snoop.py
Test: sudo make test-07
"""
from bcc import BPF

prog = r"""
struct data_t { u32 pid; char comm[16]; char fname[128]; };
BPF_PERF_OUTPUT(events);

TRACEPOINT_PROBE(syscalls, sys_enter_openat) {
    struct data_t d = {};
    d.pid = bpf_get_current_pid_tgid() >> 32;
    bpf_get_current_comm(&d.comm, sizeof(d.comm));
    bpf_probe_read_user_str(&d.fname, sizeof(d.fname), (void *)args->filename);
    events.perf_submit(args, &d, sizeof(d));
    return 0;
}
"""

b = BPF(text=prog)
print(f"{'PID':<8}{'COMM':<16}FILE")
def cb(cpu, data, size):
    e = b["events"].event(data)
    print(f"{e.pid:<8}{e.comm.decode('utf-8','replace'):<16}"
          f"{e.fname.decode('utf-8','replace')}")
b["events"].open_perf_buffer(cb, page_cnt=64)
try:
    while True: b.perf_buffer_poll()
except KeyboardInterrupt: pass
