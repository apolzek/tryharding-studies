#!/usr/bin/env python3
"""
36 — Tempo off-CPU por PID/comm (sleeping, blocking, waiting).

Mede intervalo entre sched_switch (saída) e sched_wakeup.

Run: sudo python3 36_offcputime_by_pid.py   (Ctrl-C)
Test: sudo make test-36
"""
from bcc import BPF

prog = r"""
BPF_HASH(start, u32, u64);
struct k_t { u32 pid; char comm[16]; };
BPF_HASH(total, struct k_t, u64);

TRACEPOINT_PROBE(sched, sched_switch) {
    u64 t = bpf_ktime_get_ns();
    u32 prev = args->prev_pid;
    start.update(&prev, &t);

    u32 next = args->next_pid;
    u64 *s = start.lookup(&next);
    if (s) {
        u64 delta = t - *s;
        start.delete(&next);
        struct k_t k = { .pid = next };
        __builtin_memcpy(k.comm, args->next_comm, 16);
        u64 z = 0, *v = total.lookup_or_try_init(&k, &z);
        if (v) __sync_fetch_and_add(v, delta);
    }
    return 0;
}
"""

b = BPF(text=prog)
print("tracking off-CPU per pid. Ctrl-C")
try:
    while True: pass
except KeyboardInterrupt: pass

rows = sorted(((k.pid, k.comm.decode('utf-8','replace'), v.value)
               for k, v in b["total"].items()), key=lambda r: -r[2])
print(f"\n{'PID':<8}{'COMM':<20}{'OFFCPU_MS':>12}")
for p, c, ns in rows[:25]:
    print(f"{p:<8}{c:<20}{ns/1e6:>12.1f}")
