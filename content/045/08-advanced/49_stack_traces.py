#!/usr/bin/env python3
"""
49 — Top kernel stacks que entram em uma função (usa BPF_STACK_TRACE).

Uso: sudo python3 49_stack_traces.py kfree_skb 5
"""
import sys
from bcc import BPF

fn = sys.argv[1] if len(sys.argv) > 1 else "kfree_skb"
dur = int(sys.argv[2]) if len(sys.argv) > 2 else 5

prog = r"""
BPF_STACK_TRACE(stacks, 2048);
BPF_HASH(cnt, int, u64);

int kp(struct pt_regs *ctx) {
    int id = stacks.get_stackid(ctx, 0);
    u64 z = 0, *v = cnt.lookup_or_try_init(&id, &z);
    if (v) __sync_fetch_and_add(v, 1);
    return 0;
}
"""

b = BPF(text=prog)
b.attach_kprobe(event=fn, fn_name="kp")
import time; time.sleep(dur)

stacks = b["stacks"]
rows = sorted(b["cnt"].items(), key=lambda kv: -kv[1].value)[:10]
for k, v in rows:
    print(f"\n-- {v.value} samples --")
    for addr in stacks.walk(k.value):
        print(f"  {b.ksym(addr).decode('utf-8','replace')}")
