#!/usr/bin/env python3
"""
45 — uprobe em /bin/bash:readline → captura cada comando digitado.

Demonstra uprobe em função de userspace e retorno de string (RC aponta
para buffer alocado pelo readline).

Run: sudo python3 45_bash_readline.py
Test: sudo make test-45
"""
from bcc import BPF

prog = r"""
struct data_t { u32 pid; char line[120]; };
BPF_PERF_OUTPUT(events);

int urp(struct pt_regs *ctx) {
    struct data_t d = {};
    d.pid = bpf_get_current_pid_tgid() >> 32;
    bpf_probe_read_user(&d.line, sizeof(d.line), (void *)PT_REGS_RC(ctx));
    events.perf_submit(ctx, &d, sizeof(d));
    return 0;
}
"""

b = BPF(text=prog)
b.attach_uretprobe(name="/bin/bash", sym="readline", fn_name="urp")
print(f"{'PID':<8}LINE")
def cb(cpu, data, size):
    e = b["events"].event(data)
    print(f"{e.pid:<8}{e.line.decode('utf-8','replace').rstrip()}")
b["events"].open_perf_buffer(cb)
try:
    while True: b.perf_buffer_poll()
except KeyboardInterrupt: pass
