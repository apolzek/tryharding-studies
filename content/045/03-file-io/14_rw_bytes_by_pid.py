#!/usr/bin/env python3
"""
14 — Bytes lidos/escritos por PID via vfs_read/vfs_write.

Kprobe em camada VFS pega todos os FS (ext4, tmpfs, procfs, ...).

Run: sudo python3 14_rw_bytes_by_pid.py   (Ctrl-C para ranking)
Test: sudo make test-14
"""
from bcc import BPF

prog = r"""
struct stat_t { u64 rbytes; u64 wbytes; char comm[16]; };
BPF_HASH(stats, u32, struct stat_t);

static void bump(u32 pid, u64 n, int is_write) {
    struct stat_t zero = {};
    struct stat_t *s = stats.lookup_or_try_init(&pid, &zero);
    if (!s) return;
    bpf_get_current_comm(&s->comm, sizeof(s->comm));
    if (is_write) __sync_fetch_and_add(&s->wbytes, n);
    else          __sync_fetch_and_add(&s->rbytes, n);
}

int kp_read(struct pt_regs *ctx, void *file, void *buf, size_t count) {
    bump(bpf_get_current_pid_tgid() >> 32, count, 0); return 0;
}
int kp_write(struct pt_regs *ctx, void *file, void *buf, size_t count) {
    bump(bpf_get_current_pid_tgid() >> 32, count, 1); return 0;
}
"""

b = BPF(text=prog)
b.attach_kprobe(event="vfs_read",  fn_name="kp_read")
b.attach_kprobe(event="vfs_write", fn_name="kp_write")

print("tracking VFS r/w bytes per pid... Ctrl-C to stop")
try:
    while True: pass
except KeyboardInterrupt: pass

rows = [(k.value, v.comm.decode('utf-8','replace'), v.rbytes, v.wbytes)
        for k, v in b["stats"].items()]
rows.sort(key=lambda r: -(r[2] + r[3]))
print(f"\n{'PID':<8}{'COMM':<16}{'READ':>14}{'WRITE':>14}")
for pid, comm, r, w in rows[:25]:
    print(f"{pid:<8}{comm:<16}{r:>14}{w:>14}")
