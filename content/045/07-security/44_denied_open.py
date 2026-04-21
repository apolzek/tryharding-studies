#!/usr/bin/env python3
"""
44 — open() que retornou EACCES/EPERM: "quem não está conseguindo abrir quê".

Ótimo para diagnosticar permissões quebradas em containers.

Run: sudo python3 44_denied_open.py
Test: sudo make test-44
"""
from bcc import BPF
import errno

prog = r"""
struct data_t { u32 pid; s32 ret; char comm[16]; char fname[128]; };
BPF_HASH(paths, u32, u64);
BPF_PERF_OUTPUT(events);

TRACEPOINT_PROBE(syscalls, sys_enter_openat) {
    u32 tid = bpf_get_current_pid_tgid();
    u64 p = (u64)args->filename;
    paths.update(&tid, &p);
    return 0;
}
TRACEPOINT_PROBE(syscalls, sys_exit_openat) {
    if (args->ret >= 0) return 0;
    u32 tid = bpf_get_current_pid_tgid();
    u64 *pp = paths.lookup(&tid);
    if (!pp) return 0;
    struct data_t d = {};
    d.pid = tid >> 0;
    d.pid = bpf_get_current_pid_tgid() >> 32;
    d.ret = args->ret;
    bpf_get_current_comm(&d.comm, sizeof(d.comm));
    bpf_probe_read_user_str(&d.fname, sizeof(d.fname), (void *)*pp);
    paths.delete(&tid);
    events.perf_submit(args, &d, sizeof(d));
    return 0;
}
"""

b = BPF(text=prog)
print(f"{'PID':<8}{'COMM':<16}{'ERR':<12}FILE")
def cb(cpu, data, size):
    e = b["events"].event(data)
    err = errno.errorcode.get(-e.ret, str(-e.ret))
    print(f"{e.pid:<8}{e.comm.decode():<16}{err:<12}"
          f"{e.fname.decode('utf-8','replace')}")
b["events"].open_perf_buffer(cb, page_cnt=64)
try:
    while True: b.perf_buffer_poll()
except KeyboardInterrupt: pass
