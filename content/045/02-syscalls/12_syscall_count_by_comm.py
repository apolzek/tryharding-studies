#!/usr/bin/env python3
"""
12 — Ranking: total de syscalls por comando (top N no Ctrl-C).

Usa raw_syscalls:sys_enter → captura *todas* as syscalls.

Run: sudo python3 12_syscall_count_by_comm.py
Test: sudo make test-12
"""
from bcc import BPF

prog = r"""
struct key_t { char comm[16]; };
BPF_HASH(cnt, struct key_t, u64);

TRACEPOINT_PROBE(raw_syscalls, sys_enter) {
    struct key_t k = {};
    bpf_get_current_comm(&k.comm, sizeof(k.comm));
    u64 zero = 0, *v = cnt.lookup_or_try_init(&k, &zero);
    if (v) __sync_fetch_and_add(v, 1);
    return 0;
}
"""

b = BPF(text=prog)
print("counting syscalls per comm... Ctrl-C for ranking")
try:
    while True: pass
except KeyboardInterrupt: pass

items = sorted(b["cnt"].items(), key=lambda kv: -kv[1].value)[:25]
print(f"\n{'COMM':<20}{'SYSCALLS':>12}")
for k, v in items:
    print(f"{k.comm.decode('utf-8','replace'):<20}{v.value:>12}")
