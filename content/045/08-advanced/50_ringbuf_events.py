#!/usr/bin/env python3
"""
50 — BPF ring buffer (kernel ≥ 5.8): mais rápido e simples que perf buffer.

Um único buffer global (não per-CPU), reserve/submit ao invés de copy.

Run: sudo python3 50_ringbuf_events.py
Test: sudo make test-50
"""
from bcc import BPF

prog = r"""
struct ev_t { u32 pid; char comm[16]; };
BPF_RINGBUF_OUTPUT(events, 8);

TRACEPOINT_PROBE(sched, sched_process_exec) {
    struct ev_t *e = events.ringbuf_reserve(sizeof(*e));
    if (!e) return 0;
    e->pid = bpf_get_current_pid_tgid() >> 32;
    bpf_get_current_comm(&e->comm, sizeof(e->comm));
    events.ringbuf_submit(e, 0);
    return 0;
}
"""

b = BPF(text=prog)
print(f"{'PID':<8}COMM")
def cb(ctx, data, size):
    e = b["events"].event(data)
    print(f"{e.pid:<8}{e.comm.decode('utf-8','replace')}")
b["events"].open_ring_buffer(cb)
try:
    while True: b.ring_buffer_poll()
except KeyboardInterrupt: pass
