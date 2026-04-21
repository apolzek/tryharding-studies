#!/usr/bin/env python3
"""
46 — Captura plaintext de conexões TLS via uprobe em SSL_write/SSL_read
     do libssl (OpenSSL). Funciona com curl, wget, python requests, ...

Necessário: libssl.so.3 presente (Ubuntu 24.04 default).

Run:     sudo python3 46_ssl_snoop.py
Test: sudo make test-46
Trigger: curl https://example.com
"""
import glob
from bcc import BPF

libssl = (glob.glob("/usr/lib/*/libssl.so.3") or
          glob.glob("/lib/*/libssl.so.3"))[0]

prog = r"""
struct data_t { u32 pid; int op; int len; char data[256]; char comm[16]; };
BPF_PERF_OUTPUT(events);

static void emit(struct pt_regs *ctx, int op, const void *buf, int len) {
    struct data_t d = {};
    d.pid = bpf_get_current_pid_tgid() >> 32;
    d.op = op;
    d.len = len;
    bpf_get_current_comm(&d.comm, sizeof(d.comm));
    if (len > (int)sizeof(d.data)) len = sizeof(d.data);
    if (len > 0) bpf_probe_read_user(&d.data, len, buf);
    events.perf_submit(ctx, &d, sizeof(d));
}
int up_write(struct pt_regs *ctx, void *ssl, const void *buf, int len) {
    emit(ctx, 'W', buf, len); return 0;
}
int up_read(struct pt_regs *ctx, void *ssl, void *buf, int len) {
    emit(ctx, 'R', buf, len); return 0;
}
"""

b = BPF(text=prog)
b.attach_uprobe(name=libssl, sym="SSL_write", fn_name="up_write")
b.attach_uprobe(name=libssl, sym="SSL_read",  fn_name="up_read")

print(f"{'PID':<8}{'COMM':<14}{'OP':<4}LEN  DATA...")
def cb(cpu, data, size):
    e = b["events"].event(data)
    txt = bytes(e.data)[:e.len].decode('utf-8','replace').replace('\n','\\n')
    print(f"{e.pid:<8}{e.comm.decode():<14}{chr(e.op):<4}{e.len:<5}{txt[:120]}")
b["events"].open_perf_buffer(cb, page_cnt=64)
try:
    while True: b.perf_buffer_poll()
except KeyboardInterrupt: pass
