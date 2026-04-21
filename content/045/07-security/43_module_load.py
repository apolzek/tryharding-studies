#!/usr/bin/env python3
"""
43 — Detecta carga de módulos de kernel (init_module / finit_module).

Attack surface clássica — carregar módulo ruim = rootkit.

Run: sudo python3 43_module_load.py
Test: sudo make test-43
"""
from bcc import BPF

prog = r"""
struct data_t { u32 pid; u32 uid; char comm[16]; char mod[64]; };
BPF_PERF_OUTPUT(events);

int kp(struct pt_regs *ctx, struct module *mod) {
    struct data_t d = {};
    d.pid = bpf_get_current_pid_tgid() >> 32;
    d.uid = bpf_get_current_uid_gid();
    bpf_get_current_comm(&d.comm, sizeof(d.comm));
    bpf_probe_read_kernel_str(&d.mod, sizeof(d.mod), mod->name);
    events.perf_submit(ctx, &d, sizeof(d));
    return 0;
}
"""

b = BPF(text=prog)
b.attach_kprobe(event="do_init_module", fn_name="kp")
print(f"{'PID':<8}{'UID':<6}{'COMM':<16}MODULE")
def cb(cpu, data, size):
    e = b["events"].event(data)
    print(f"{e.pid:<8}{e.uid:<6}{e.comm.decode():<16}"
          f"{e.mod.decode('utf-8','replace')}")
b["events"].open_perf_buffer(cb)
try:
    while True: b.perf_buffer_poll()
except KeyboardInterrupt: pass
