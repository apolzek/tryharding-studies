#!/usr/bin/env python3
"""
32 — Bytes TCP enviados por PID (tcp_sendmsg).

Run: sudo python3 32_tcp_send_bytes_by_pid.py
Test: sudo make test-32
"""
from bcc import BPF

prog = r"""
BPF_HASH(st, u32, u64);

int kp(struct pt_regs *ctx, struct sock *sk, struct msghdr *msg, size_t len) {
    u32 pid = bpf_get_current_pid_tgid() >> 32;
    u64 z = 0, *v = st.lookup_or_try_init(&pid, &z);
    if (v) __sync_fetch_and_add(v, len);
    return 0;
}
"""

b = BPF(text=prog)
b.attach_kprobe(event="tcp_sendmsg", fn_name="kp")
print("tracking tcp bytes sent per pid... Ctrl-C")
try:
    while True: pass
except KeyboardInterrupt: pass

rows = sorted(((k.value, v.value) for k, v in b["st"].items()), key=lambda r: -r[1])
print(f"\n{'PID':<8}{'BYTES':>14}")
for p, by in rows[:25]:
    print(f"{p:<8}{by:>14}")
