#!/usr/bin/env python3
"""
21 — Árvore de fork: mostra parent → child.

Run: sudo python3 21_fork_tree.py
Test: sudo make test-21
"""
from bcc import BPF

prog = r"""
struct data_t { u32 ppid; u32 child; char comm[16]; };
BPF_PERF_OUTPUT(events);

TRACEPOINT_PROBE(sched, sched_process_fork) {
    struct data_t d = { .ppid = args->parent_pid, .child = args->child_pid };
    __builtin_memcpy(d.comm, args->parent_comm, 16);
    events.perf_submit(args, &d, sizeof(d));
    return 0;
}
"""

b = BPF(text=prog)
print(f"{'PPID':<8}{'CHILD':<8}{'PARENT_COMM'}")
def cb(cpu, data, size):
    e = b["events"].event(data)
    print(f"{e.ppid:<8}{e.child:<8}{e.comm.decode('utf-8','replace')}")
b["events"].open_perf_buffer(cb)
try:
    while True: b.perf_buffer_poll()
except KeyboardInterrupt: pass
