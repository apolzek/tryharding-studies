#!/usr/bin/env python3
"""
04 — BPF_HASH: conta syscalls por PID e mostra top N no final.

Run:  sudo python3 04_hash_map.py   (Ctrl-C para ver ranking)
Test: sudo make test-04
"""
from bcc import BPF

prog = r"""
BPF_HASH(by_pid, u32, u64);

int count(void *ctx) {
    u32 pid = bpf_get_current_pid_tgid() >> 32;
    u64 zero = 0, *v;
    v = by_pid.lookup_or_try_init(&pid, &zero);
    if (v) __sync_fetch_and_add(v, 1);
    return 0;
}
"""

b = BPF(text=prog)
b.attach_kprobe(event=b.get_syscall_fnname("write"), fn_name="count")

print("counting write() per-PID. Ctrl-C for results.")
try:
    while True:
        pass
except KeyboardInterrupt:
    pass

print(f"\n{'PID':<8}{'COUNT':>10}")
items = sorted(b["by_pid"].items(), key=lambda kv: -kv[1].value)[:20]
for k, v in items:
    print(f"{k.value:<8}{v.value:>10}")
