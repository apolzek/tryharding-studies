#!/usr/bin/env python3
"""
03 — BPF_PERF_OUTPUT: enviando eventos estruturados ao userspace.

Padrão mais comum: kernel enche um struct e chama events.perf_submit();
userspace registra uma callback que é chamada para cada evento.

Run: sudo python3 03_perf_event.py   (gera tráfego fazendo ls/date)
Test: sudo make test-03
"""
from bcc import BPF

prog = r"""
struct data_t {
    u64 ts;
    u32 pid;
    char comm[16];
};
BPF_PERF_OUTPUT(events);

int on_clone(void *ctx) {
    struct data_t d = {};
    d.ts  = bpf_ktime_get_ns();
    d.pid = bpf_get_current_pid_tgid() >> 32;
    bpf_get_current_comm(&d.comm, sizeof(d.comm));
    events.perf_submit(ctx, &d, sizeof(d));
    return 0;
}
"""

b = BPF(text=prog)
b.attach_kprobe(event=b.get_syscall_fnname("clone"), fn_name="on_clone")

print(f"{'TIME(ns)':<20}{'PID':<8}{'COMM'}")
def handler(cpu, data, size):
    e = b["events"].event(data)
    print(f"{e.ts:<20}{e.pid:<8}{e.comm.decode('utf-8','replace')}")

b["events"].open_perf_buffer(handler)
try:
    while True:
        b.perf_buffer_poll()
except KeyboardInterrupt:
    pass
