#!/usr/bin/env python3
"""
40 — Snoop de setuid: quem vira quem (escalada de privilégio).

Run: sudo python3 40_setuid_snoop.py
Test: sudo make test-40
"""
from bcc import BPF

prog = r"""
struct data_t { u32 pid; u32 cur_uid; u32 new_uid; char comm[16]; };
BPF_PERF_OUTPUT(events);

TRACEPOINT_PROBE(syscalls, sys_enter_setuid) {
    struct data_t d = {};
    d.pid = bpf_get_current_pid_tgid() >> 32;
    d.cur_uid = bpf_get_current_uid_gid();
    d.new_uid = args->uid;
    bpf_get_current_comm(&d.comm, sizeof(d.comm));
    events.perf_submit(args, &d, sizeof(d));
    return 0;
}
"""

b = BPF(text=prog)
print(f"{'PID':<8}{'COMM':<16}{'CUR_UID':>8}{'NEW_UID':>8}")
def cb(cpu, data, size):
    e = b["events"].event(data)
    print(f"{e.pid:<8}{e.comm.decode():<16}{e.cur_uid:>8}{e.new_uid:>8}")
b["events"].open_perf_buffer(cb)
try:
    while True: b.perf_buffer_poll()
except KeyboardInterrupt: pass
