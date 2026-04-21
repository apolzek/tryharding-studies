#!/usr/bin/env python3
"""
39 — Rastreia cada chamada a cap_capable (checagem de capability).

Útil pra descobrir quais capabilities um binário realmente precisa
antes de remover CAP_SYS_ADMIN "por via das dúvidas".

Run: sudo python3 39_capable.py
Test: sudo make test-39
"""
from bcc import BPF

CAPS = {0:"CHOWN",1:"DAC_OVERRIDE",2:"DAC_READ_SEARCH",3:"FOWNER",4:"FSETID",
        5:"KILL",6:"SETGID",7:"SETUID",8:"SETPCAP",9:"LINUX_IMMUTABLE",
        10:"NET_BIND_SERVICE",12:"NET_ADMIN",13:"NET_RAW",14:"IPC_LOCK",
        17:"SYS_RAWIO",18:"SYS_CHROOT",19:"SYS_PTRACE",21:"SYS_ADMIN",
        22:"SYS_BOOT",23:"SYS_NICE",24:"SYS_RESOURCE",25:"SYS_TIME",
        38:"BPF",39:"PERFMON"}

prog = r"""
struct data_t { u32 pid; u32 uid; int cap; char comm[16]; };
BPF_PERF_OUTPUT(events);

int kp(struct pt_regs *ctx, const struct cred *cred, struct user_namespace *ns,
       int cap, unsigned int opts) {
    struct data_t d = {};
    d.pid = bpf_get_current_pid_tgid() >> 32;
    d.uid = bpf_get_current_uid_gid();
    d.cap = cap;
    bpf_get_current_comm(&d.comm, sizeof(d.comm));
    events.perf_submit(ctx, &d, sizeof(d));
    return 0;
}
"""

b = BPF(text=prog)
b.attach_kprobe(event="cap_capable", fn_name="kp")
print(f"{'PID':<8}{'UID':<6}{'COMM':<16}CAP")
def cb(cpu, data, size):
    e = b["events"].event(data)
    print(f"{e.pid:<8}{e.uid:<6}{e.comm.decode():<16}"
          f"{CAPS.get(e.cap,f'?{e.cap}')}")
b["events"].open_perf_buffer(cb)
try:
    while True: b.perf_buffer_poll()
except KeyboardInterrupt: pass
