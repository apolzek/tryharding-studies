#!/usr/bin/env python3
"""
16 — Alerta quando uma vfs_read demora mais que THRESH_US.

Run: sudo python3 16_slow_vfs_read.py
Test: sudo make test-16
"""
from bcc import BPF

prog = r"""
#define THRESH_US 1000
BPF_HASH(start, u32, u64);
struct ev_t { u32 pid; u64 us; char comm[16]; };
BPF_PERF_OUTPUT(events);

int kp_read(struct pt_regs *ctx) {
    u32 tid = bpf_get_current_pid_tgid();
    u64 t = bpf_ktime_get_ns();
    start.update(&tid, &t);
    return 0;
}
int krp_read(struct pt_regs *ctx) {
    u32 tid = bpf_get_current_pid_tgid();
    u64 *t = start.lookup(&tid);
    if (!t) return 0;
    u64 dt = (bpf_ktime_get_ns() - *t) / 1000;
    start.delete(&tid);
    if (dt < THRESH_US) return 0;
    struct ev_t e = { .pid = tid >> 0, .us = dt };
    e.pid = bpf_get_current_pid_tgid() >> 32;
    bpf_get_current_comm(&e.comm, sizeof(e.comm));
    events.perf_submit(ctx, &e, sizeof(e));
    return 0;
}
"""

b = BPF(text=prog)
b.attach_kprobe(event="vfs_read",    fn_name="kp_read")
b.attach_kretprobe(event="vfs_read", fn_name="krp_read")
print(f"{'PID':<8}{'COMM':<16}{'us':>10}")
def cb(cpu, data, size):
    e = b["events"].event(data)
    print(f"{e.pid:<8}{e.comm.decode():<16}{e.us:>10}")
b["events"].open_perf_buffer(cb)
try:
    while True: b.perf_buffer_poll()
except KeyboardInterrupt: pass
