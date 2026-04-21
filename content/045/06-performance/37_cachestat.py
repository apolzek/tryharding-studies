#!/usr/bin/env python3
"""
37 — Page cache stats: hit/miss ratio (agregado do sistema).

Kprobes em mark_page_accessed / mark_buffer_dirty / add_to_page_cache_lru
/ account_page_dirtied. Mesmo algoritmo do cachestat original do BCC.

Run: sudo python3 37_cachestat.py    (imprime a cada 1s)
Test: sudo make test-37
"""
import time
from bcc import BPF

prog = r"""
BPF_HASH(c, u32, u64);

static void bump(u32 idx) {
    u64 z = 0, *v = c.lookup_or_try_init(&idx, &z);
    if (v) __sync_fetch_and_add(v, 1);
}
int p_mpa(struct pt_regs *ctx)      { bump(0); return 0; }  /* accesses */
int p_mbd(struct pt_regs *ctx)      { bump(1); return 0; }  /* dirty */
int p_add(struct pt_regs *ctx)      { bump(2); return 0; }  /* added */
int p_apd(struct pt_regs *ctx)      { bump(3); return 0; }  /* accounted */
"""
b = BPF(text=prog)
try:
    b.attach_kprobe(event="mark_page_accessed",      fn_name="p_mpa")
    b.attach_kprobe(event="mark_buffer_dirty",       fn_name="p_mbd")
    b.attach_kprobe(event="add_to_page_cache_lru",   fn_name="p_add")
    b.attach_kprobe(event="account_page_dirtied",    fn_name="p_apd")
except Exception as e:
    print(f"attach failed ({e}) — kernel symbol names may differ; try bpftool prog")
    raise

prev = {i: 0 for i in range(4)}
print(f"{'HITS':>10}{'MISSES':>10}{'DIRTIES':>10}{'HIT%':>8}")
try:
    while True:
        time.sleep(1)
        cur = {i: (b["c"][i].value if i in [k.value for k in b["c"].keys()] else 0)
               for i in range(4)}
        mpa, mbd, add, apd = (cur[i] - prev[i] for i in range(4))
        prev = cur
        misses  = max(0, add - mbd)
        accesses = mpa
        hits = max(0, accesses - misses)
        dirties = apd
        ratio = (hits * 100.0 / (hits + misses)) if (hits + misses) else 0
        print(f"{hits:>10}{misses:>10}{dirties:>10}{ratio:>8.1f}")
except KeyboardInterrupt: pass
