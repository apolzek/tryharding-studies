#!/usr/bin/env python3
"""
15 — Abre − fecha = FDs possivelmente vazando. Por processo.

Não pega tudo (dup, fcntl, etc.) — mas é ótima aproximação inicial.

Run: sudo python3 15_fd_usage.py
Test: sudo make test-15
"""
from bcc import BPF

prog = r"""
BPF_HASH(opens,  u32, u64);
BPF_HASH(closes, u32, u64);

TRACEPOINT_PROBE(syscalls, sys_exit_openat) {
    if (args->ret < 0) return 0;
    u32 pid = bpf_get_current_pid_tgid() >> 32;
    u64 z = 0, *v = opens.lookup_or_try_init(&pid, &z);
    if (v) __sync_fetch_and_add(v, 1);
    return 0;
}
TRACEPOINT_PROBE(syscalls, sys_enter_close) {
    u32 pid = bpf_get_current_pid_tgid() >> 32;
    u64 z = 0, *v = closes.lookup_or_try_init(&pid, &z);
    if (v) __sync_fetch_and_add(v, 1);
    return 0;
}
"""

b = BPF(text=prog)
print("accounting open/close per pid... Ctrl-C for ranking")
try:
    while True: pass
except KeyboardInterrupt: pass

op = {k.value: v.value for k, v in b["opens"].items()}
cl = {k.value: v.value for k, v in b["closes"].items()}
pids = sorted(set(op) | set(cl), key=lambda p: -(op.get(p, 0) - cl.get(p, 0)))
print(f"\n{'PID':<8}{'OPEN':>8}{'CLOSE':>8}{'DELTA':>8}")
for p in pids[:25]:
    o, c = op.get(p, 0), cl.get(p, 0)
    print(f"{p:<8}{o:>8}{c:>8}{o-c:>8}")
