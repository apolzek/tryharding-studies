#!/usr/bin/env python3
"""
41 — Snoop de mount(2): container escapes costumam passar por aqui.

Run: sudo python3 41_mount_snoop.py
Test: sudo make test-41
"""
from bcc import BPF

prog = r"""
struct data_t { u32 pid; char comm[16]; char src[96]; char dst[96]; char fs[32]; };
BPF_PERF_OUTPUT(events);

TRACEPOINT_PROBE(syscalls, sys_enter_mount) {
    struct data_t d = {};
    d.pid = bpf_get_current_pid_tgid() >> 32;
    bpf_get_current_comm(&d.comm, sizeof(d.comm));
    bpf_probe_read_user_str(&d.src, sizeof(d.src), (void *)args->dev_name);
    bpf_probe_read_user_str(&d.dst, sizeof(d.dst), (void *)args->dir_name);
    bpf_probe_read_user_str(&d.fs,  sizeof(d.fs),  (void *)args->type);
    events.perf_submit(args, &d, sizeof(d));
    return 0;
}
"""

b = BPF(text=prog)
print(f"{'PID':<8}{'COMM':<14}{'FS':<10}SRC -> DST")
def cb(cpu, data, size):
    e = b["events"].event(data)
    print(f"{e.pid:<8}{e.comm.decode():<14}{e.fs.decode('utf-8','replace'):<10}"
          f"{e.src.decode('utf-8','replace')} -> {e.dst.decode('utf-8','replace')}")
b["events"].open_perf_buffer(cb)
try:
    while True: b.perf_buffer_poll()
except KeyboardInterrupt: pass
