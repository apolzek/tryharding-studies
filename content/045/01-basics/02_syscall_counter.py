#!/usr/bin/env python3
"""
02 — Contador global de uma syscall usando BPF_ARRAY.

Mostra como ler um map do userspace periodicamente.

Run: sudo python3 02_syscall_counter.py
Test: sudo make test-02
"""
import time
from bcc import BPF

prog = r"""
BPF_ARRAY(counter, u64, 1);

int count_call(void *ctx) {
    u32 key = 0;
    u64 *v = counter.lookup(&key);
    if (v) __sync_fetch_and_add(v, 1);
    return 0;
}
"""

b = BPF(text=prog)
b.attach_kprobe(event=b.get_syscall_fnname("openat"), fn_name="count_call")

print("counting openat() calls, Ctrl-C to stop")
try:
    while True:
        time.sleep(1)
        v = b["counter"][0].value
        print(f"openat total: {v}")
except KeyboardInterrupt:
    pass
