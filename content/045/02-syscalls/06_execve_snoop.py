#!/usr/bin/env python3
"""
06 — execve snoop: cada comando novo no sistema (com PID e comm).

Anexa no tracepoint sched_process_exec (mais estável que kprobe).

Run: sudo python3 06_execve_snoop.py
Test: sudo make test-06
"""
from bcc import BPF

prog = r"""
struct data_t { u32 pid; u32 ppid; char comm[16]; };
BPF_PERF_OUTPUT(events);

TRACEPOINT_PROBE(sched, sched_process_exec) {
    struct data_t d = {};
    d.pid  = bpf_get_current_pid_tgid() >> 32;
    struct task_struct *t = (struct task_struct *)bpf_get_current_task();
    d.ppid = t->real_parent->tgid;
    bpf_get_current_comm(&d.comm, sizeof(d.comm));
    events.perf_submit(args, &d, sizeof(d));
    return 0;
}
"""

b = BPF(text=prog)
print(f"{'PID':<8}{'PPID':<8}COMM")
def cb(cpu, data, size):
    e = b["events"].event(data)
    print(f"{e.pid:<8}{e.ppid:<8}{e.comm.decode('utf-8','replace')}")
b["events"].open_perf_buffer(cb)
try:
    while True: b.perf_buffer_poll()
except KeyboardInterrupt: pass
