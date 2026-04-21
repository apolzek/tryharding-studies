#!/usr/bin/env python3
"""
34 — Biosnoop: cada I/O de bloco com device, offset, tamanho, latência.

Run: sudo python3 34_biosnoop.py
Test: sudo make test-34
"""
from bcc import BPF

prog = r"""
BPF_HASH(start, u64, u64);
struct ev_t { u32 major, minor, pid; u64 sector, nr_sector, us; char comm[16]; char rwbs[8]; };
BPF_PERF_OUTPUT(events);

TRACEPOINT_PROBE(block, block_rq_issue) {
    u64 rq = (u64)args->dev;
    u64 ts = bpf_ktime_get_ns();
    start.update(&rq, &ts);
    return 0;
}
TRACEPOINT_PROBE(block, block_rq_complete) {
    u64 rq = (u64)args->dev;
    u64 *ts = start.lookup(&rq); if (!ts) return 0;
    u64 dt = (bpf_ktime_get_ns() - *ts) / 1000;
    start.delete(&rq);
    struct ev_t e = {};
    e.major = args->dev >> 20;
    e.minor = args->dev & ((1 << 20) - 1);
    e.sector = args->sector;
    e.nr_sector = args->nr_sector;
    e.us = dt;
    e.pid = bpf_get_current_pid_tgid() >> 32;
    bpf_get_current_comm(&e.comm, sizeof(e.comm));
    __builtin_memcpy(e.rwbs, args->rwbs, sizeof(e.rwbs));
    events.perf_submit(args, &e, sizeof(e));
    return 0;
}
"""

b = BPF(text=prog)
print(f"{'PID':<8}{'COMM':<16}{'DEV':<10}{'OP':<5}{'SECTOR':>14}{'nSEC':>8}{'us':>10}")
def cb(cpu, data, size):
    e = b["events"].event(data)
    print(f"{e.pid:<8}{e.comm.decode():<16}{e.major}:{e.minor:<8}"
          f"{e.rwbs.decode('ascii','replace').strip(chr(0)):<5}"
          f"{e.sector:>14}{e.nr_sector:>8}{e.us:>10}")
b["events"].open_perf_buffer(cb)
try:
    while True: b.perf_buffer_poll()
except KeyboardInterrupt: pass
