#!/usr/bin/env python3
"""
01 — Hello World em eBPF.

Anexa um kprobe na syscall clone: toda vez que um processo é criado
(fork/exec), o kernel imprime uma linha no trace_pipe.

Run:   sudo python3 01_hello.py
Test: sudo make test-01
Trigger: em outro terminal, rode qualquer comando (ls, date, …).
Stop:  Ctrl-C.
"""
from bcc import BPF

prog = r"""
int hello(void *ctx) {
    bpf_trace_printk("hello, ebpf world!\n");
    return 0;
}
"""

b = BPF(text=prog)
b.attach_kprobe(event=b.get_syscall_fnname("clone"), fn_name="hello")

print("tracing clone()... Ctrl-C to stop")
print(f"{'TIME':<18}{'PID':<8}{'COMM':<16}MSG")
try:
    b.trace_print()
except KeyboardInterrupt:
    pass
