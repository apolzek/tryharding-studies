#!/usr/bin/env python3
"""
17 — Conta vfs_read/write por inode->i_sb->s_type->name (fs type).

Mostra "quem faz I/O em ext4 vs tmpfs vs procfs".

Run: sudo python3 17_fs_ops_by_fstype.py
Test: sudo make test-17
"""
from bcc import BPF

prog = r"""
#include <linux/fs.h>
struct key_t { char fs[16]; char op; };
BPF_HASH(cnt, struct key_t, u64);

static void bump(struct file *f, char op) {
    if (!f) return;
    struct key_t k = { .op = op };
    struct super_block *sb = f->f_inode->i_sb;
    const char *n = sb->s_type->name;
    bpf_probe_read_kernel_str(&k.fs, sizeof(k.fs), n);
    u64 z = 0, *v = cnt.lookup_or_try_init(&k, &z);
    if (v) __sync_fetch_and_add(v, 1);
}

int kp_read(struct pt_regs *ctx, struct file *f) { bump(f, 'R'); return 0; }
int kp_write(struct pt_regs *ctx, struct file *f) { bump(f, 'W'); return 0; }
"""

b = BPF(text=prog)
b.attach_kprobe(event="vfs_read",  fn_name="kp_read")
b.attach_kprobe(event="vfs_write", fn_name="kp_write")

print("counting vfs ops per fstype... Ctrl-C")
try:
    while True: pass
except KeyboardInterrupt: pass

rows = [(k.fs.decode('utf-8','replace'), chr(k.op), v.value) for k, v in b["cnt"].items()]
rows.sort(key=lambda r: -r[2])
print(f"\n{'FS':<12}{'OP':<4}{'COUNT':>10}")
for fs, op, c in rows:
    print(f"{fs:<12}{op:<4}{c:>10}")
