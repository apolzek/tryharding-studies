#!/usr/bin/env python3
"""
38 — Contagem de softirqs por tipo (NET_RX, TIMER, TASKLET, ...).

Run: sudo python3 38_softirqs.py   (Ctrl-C)
Test: sudo make test-38
"""
from bcc import BPF

VECTORS = ["HI","TIMER","NET_TX","NET_RX","BLOCK","IRQ_POLL",
           "TASKLET","SCHED","HRTIMER","RCU"]

prog = r"""
BPF_HASH(cnt, u32, u64);

TRACEPOINT_PROBE(irq, softirq_entry) {
    u32 v = args->vec;
    u64 z = 0, *p = cnt.lookup_or_try_init(&v, &z);
    if (p) __sync_fetch_and_add(p, 1);
    return 0;
}
"""

b = BPF(text=prog)
print("counting softirqs... Ctrl-C")
try:
    while True: pass
except KeyboardInterrupt: pass

rows = [(k.value, v.value) for k, v in b["cnt"].items()]
rows.sort(key=lambda r: -r[1])
print(f"\n{'VEC':<10}{'NAME':<12}{'COUNT':>10}")
for vec, n in rows:
    name = VECTORS[vec] if vec < len(VECTORS) else f"?{vec}"
    print(f"{vec:<10}{name:<12}{n:>10}")
