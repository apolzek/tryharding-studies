#!/usr/bin/env python3
"""
22 — OOM killer snoop: quem foi morto, por quem, pontuação.

Run: sudo python3 22_oom_kill.py
Test: sudo make test-22
"""
from bcc import BPF

prog = r"""
#include <linux/oom.h>
struct data_t { u32 fpid; u32 tpid; u64 pages; char fcomm[16]; char tcomm[16]; };
BPF_PERF_OUTPUT(events);

int kp_oom(struct pt_regs *ctx, struct oom_control *oc) {
    struct data_t d = {};
    d.fpid = bpf_get_current_pid_tgid() >> 32;
    bpf_get_current_comm(&d.fcomm, sizeof(d.fcomm));
    d.tpid = oc->chosen->tgid;
    bpf_probe_read_kernel_str(&d.tcomm, sizeof(d.tcomm), oc->chosen->comm);
    d.pages = oc->totalpages;
    events.perf_submit(ctx, &d, sizeof(d));
    return 0;
}
"""

b = BPF(text=prog)
b.attach_kprobe(event="oom_kill_process", fn_name="kp_oom")
print(f"{'KILLER':<8}{'KCOMM':<16}{'VICTIM':<8}{'VCOMM':<16}TOTAL_PAGES")
def cb(cpu, data, size):
    e = b["events"].event(data)
    print(f"{e.fpid:<8}{e.fcomm.decode():<16}{e.tpid:<8}"
          f"{e.tcomm.decode():<16}{e.pages}")
b["events"].open_perf_buffer(cb)
try:
    while True: b.perf_buffer_poll()
except KeyboardInterrupt: pass
