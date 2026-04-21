#!/usr/bin/env python3
"""
47 — USDT: probe em Python interpretado (function__entry).

Ativar USDT no Python: `python3 -X dev -X pybc ...` — ou use um
interpretador compilado com --with-dtrace. O mais prático:
rode este script mesmo com Python do sistema, e use um alvo Python
que tenha USDT (Ubuntu 24.04 empacota libpython3.12 com USDT).

Run:     sudo python3 47_python_calls.py <PID>
Test: sudo make test-47
"""
import sys
from bcc import BPF, USDT

if len(sys.argv) < 2:
    sys.exit("usage: 47_python_calls.py <PID>")
pid = int(sys.argv[1])

usdt = USDT(pid=pid)
usdt.enable_probe(probe="function__entry", fn_name="do_entry")

prog = r"""
struct data_t { u32 pid; char file[80]; char fn[64]; int line; };
BPF_PERF_OUTPUT(events);

int do_entry(struct pt_regs *ctx) {
    struct data_t d = {};
    d.pid = bpf_get_current_pid_tgid() >> 32;
    uint64_t a0, a1, a2;
    bpf_usdt_readarg(1, ctx, &a0);
    bpf_usdt_readarg(2, ctx, &a1);
    bpf_usdt_readarg(3, ctx, &a2);
    bpf_probe_read_user_str(&d.file, sizeof(d.file), (void *)a0);
    bpf_probe_read_user_str(&d.fn,   sizeof(d.fn),   (void *)a1);
    d.line = (int)a2;
    events.perf_submit(ctx, &d, sizeof(d));
    return 0;
}
"""

b = BPF(text=prog, usdt_contexts=[usdt])
print(f"{'PID':<8}{'FILE:LINE':<40}FN")
def cb(cpu, data, size):
    e = b["events"].event(data)
    loc = f"{e.file.decode('utf-8','replace')}:{e.line}"
    print(f"{e.pid:<8}{loc:<40}{e.fn.decode('utf-8','replace')}")
b["events"].open_perf_buffer(cb)
try:
    while True: b.perf_buffer_poll()
except KeyboardInterrupt: pass
