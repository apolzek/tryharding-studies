#!/usr/bin/env python3
"""
19 — execve snoop COM argv. Mais elaborado: lê args->argv[i].

Passa args via BPF_PERCPU_ARRAY (stack do BPF é limitado a 512B).

Run: sudo python3 19_execsnoop_args.py
Test: sudo make test-19
"""
from bcc import BPF
import ctypes as ct

prog = r"""
#define ARGSIZE 256
enum evt { EV_ARG, EV_RET };
struct data_t {
    u32 pid; u32 ppid; enum evt type;
    int retval;
    char comm[16]; char arg[ARGSIZE];
};
BPF_PERF_OUTPUT(events);
BPF_PERCPU_ARRAY(scratch, struct data_t, 1);

static int submit_arg(void *ctx, const char __user *ptr, struct data_t *d) {
    bpf_probe_read_user_str(d->arg, sizeof(d->arg), ptr);
    events.perf_submit(ctx, d, sizeof(*d));
    return 0;
}

int syscall__execve(struct pt_regs *ctx, const char __user *filename,
                    const char __user *const __user *argv,
                    const char __user *const __user *envp) {
    u32 z = 0;
    struct data_t *d = scratch.lookup(&z);
    if (!d) return 0;
    __builtin_memset(d, 0, sizeof(*d));
    d->pid  = bpf_get_current_pid_tgid() >> 32;
    struct task_struct *t = (struct task_struct *)bpf_get_current_task();
    d->ppid = t->real_parent->tgid;
    d->type = EV_ARG;
    bpf_get_current_comm(&d->comm, sizeof(d->comm));
    submit_arg(ctx, filename, d);
    #pragma unroll
    for (int i = 1; i < 20; i++) {
        const char __user *a = NULL;
        bpf_probe_read_user(&a, sizeof(a), &argv[i]);
        if (!a) return 0;
        submit_arg(ctx, a, d);
    }
    return 0;
}

int do_ret_sys_execve(struct pt_regs *ctx) {
    u32 z = 0;
    struct data_t *d = scratch.lookup(&z);
    if (!d) return 0;
    __builtin_memset(d, 0, sizeof(*d));
    d->pid = bpf_get_current_pid_tgid() >> 32;
    d->type = EV_RET;
    d->retval = PT_REGS_RC(ctx);
    bpf_get_current_comm(&d->comm, sizeof(d->comm));
    events.perf_submit(ctx, d, sizeof(*d));
    return 0;
}
"""

b = BPF(text=prog)
exe = b.get_syscall_fnname("execve")
b.attach_kprobe(event=exe, fn_name="syscall__execve")
b.attach_kretprobe(event=exe, fn_name="do_ret_sys_execve")

argv = {}
print(f"{'PID':<8}{'PPID':<8}{'RC':<4}CMDLINE")
def cb(cpu, data, size):
    e = b["events"].event(data)
    if e.type == 0:  # EV_ARG
        argv.setdefault(e.pid, []).append(e.arg.decode('utf-8','replace'))
    else:
        parts = argv.pop(e.pid, [])
        print(f"{e.pid:<8}{0:<8}{e.retval:<4}{' '.join(parts)}")
b["events"].open_perf_buffer(cb)
try:
    while True: b.perf_buffer_poll()
except KeyboardInterrupt: pass
