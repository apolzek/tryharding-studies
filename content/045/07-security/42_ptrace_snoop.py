#!/usr/bin/env python3
"""
42 — Snoop de ptrace: debuggers, strace e... injeção em processo.

Run: sudo python3 42_ptrace_snoop.py
Test: sudo make test-42
"""
from bcc import BPF

REQ = {0:"TRACEME",1:"PEEKTEXT",2:"PEEKDATA",3:"PEEKUSER",
       4:"POKETEXT",5:"POKEDATA",6:"POKEUSER",7:"CONT",8:"KILL",9:"SINGLESTEP",
       12:"GETREGS",13:"SETREGS",16:"ATTACH",17:"DETACH",24:"SYSCALL"}

prog = r"""
struct data_t { u32 pid; s32 tpid; s32 req; char comm[16]; };
BPF_PERF_OUTPUT(events);
TRACEPOINT_PROBE(syscalls, sys_enter_ptrace) {
    struct data_t d = {};
    d.pid = bpf_get_current_pid_tgid() >> 32;
    d.tpid = args->pid;
    d.req  = args->request;
    bpf_get_current_comm(&d.comm, sizeof(d.comm));
    events.perf_submit(args, &d, sizeof(d));
    return 0;
}
"""

b = BPF(text=prog)
print(f"{'PID':<8}{'COMM':<16}{'REQ':<12}TARGET")
def cb(cpu, data, size):
    e = b["events"].event(data)
    print(f"{e.pid:<8}{e.comm.decode():<16}{REQ.get(e.req,str(e.req)):<12}{e.tpid}")
b["events"].open_perf_buffer(cb)
try:
    while True: b.perf_buffer_poll()
except KeyboardInterrupt: pass
