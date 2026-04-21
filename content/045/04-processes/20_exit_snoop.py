#!/usr/bin/env python3
"""
20 — Snoop de processos que terminaram: PID, comm, código de saída, duração.

Usa sched_process_exit e calcula start_time - exit_time.

Run: sudo python3 20_exit_snoop.py
Test: sudo make test-20
"""
from bcc import BPF

prog = r"""
#include <linux/sched.h>
struct data_t { u32 pid; u32 ppid; u32 exit_code; u64 dur_ns; char comm[16]; };
BPF_PERF_OUTPUT(events);

TRACEPOINT_PROBE(sched, sched_process_exit) {
    struct task_struct *t = (struct task_struct *)bpf_get_current_task();
    if (t->pid != t->tgid) return 0;        /* ignore threads */
    struct data_t d = {};
    d.pid  = t->tgid;
    d.ppid = t->real_parent->tgid;
    d.exit_code = t->exit_code >> 8;
    d.dur_ns = bpf_ktime_get_ns() - t->start_time;
    bpf_probe_read_kernel_str(&d.comm, sizeof(d.comm), t->comm);
    events.perf_submit(args, &d, sizeof(d));
    return 0;
}
"""

b = BPF(text=prog)
print(f"{'PID':<8}{'PPID':<8}{'EXIT':<6}{'DUR_MS':>10}  COMM")
def cb(cpu, data, size):
    e = b["events"].event(data)
    print(f"{e.pid:<8}{e.ppid:<8}{e.exit_code:<6}{e.dur_ns/1e6:>10.1f}  "
          f"{e.comm.decode('utf-8','replace')}")
b["events"].open_perf_buffer(cb)
try:
    while True: b.perf_buffer_poll()
except KeyboardInterrupt: pass
