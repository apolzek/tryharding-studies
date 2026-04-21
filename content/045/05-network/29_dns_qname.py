#!/usr/bin/env python3
"""
29 — Snoop de consultas DNS (porta 53 UDP).

Kprobe em udp_sendmsg; valida dport == 53 e lê o qname do payload.
Didático: mostra parse de DNS em eBPF (loop unrolled).

Run: sudo python3 29_dns_qname.py      (trigger: getent hosts example.com)
Test: sudo make test-29
"""
from bcc import BPF

prog = r"""
#include <net/sock.h>
#include <linux/udp.h>
#include <linux/inet.h>

struct data_t { u32 pid; char comm[16]; char qname[64]; };
BPF_PERF_OUTPUT(events);

int kp(struct pt_regs *ctx, struct sock *sk, struct msghdr *msg, size_t len) {
    u16 dport = bpf_ntohs(sk->__sk_common.skc_dport);
    if (dport != 53) return 0;
    struct data_t d = {};
    d.pid = bpf_get_current_pid_tgid() >> 32;
    bpf_get_current_comm(&d.comm, sizeof(d.comm));

    struct iov_iter *it = &msg->msg_iter;
    const struct iovec *iov = NULL;
    bpf_probe_read_kernel(&iov, sizeof(iov), &it->__iov);
    if (!iov) return 0;
    void *base = NULL; size_t iov_len = 0;
    bpf_probe_read_kernel(&base, sizeof(base), &iov->iov_base);
    bpf_probe_read_kernel(&iov_len, sizeof(iov_len), &iov->iov_len);
    if (iov_len < 13 || !base) return 0;

    /* copy qname starting at payload+12 (DNS header is 12 bytes) */
    bpf_probe_read_user(&d.qname, sizeof(d.qname), base + 12);
    events.perf_submit(ctx, &d, sizeof(d));
    return 0;
}
"""

b = BPF(text=prog)
b.attach_kprobe(event="udp_sendmsg", fn_name="kp")

def decode_qname(raw: bytes) -> str:
    out, i = [], 0
    while i < len(raw):
        n = raw[i]
        if n == 0 or n > 63: break
        i += 1
        out.append(raw[i:i+n].decode('ascii', 'replace'))
        i += n
    return ".".join(out)

print(f"{'PID':<8}{'COMM':<16}QNAME")
def cb(cpu, data, size):
    e = b["events"].event(data)
    print(f"{e.pid:<8}{e.comm.decode():<16}{decode_qname(bytes(e.qname))}")
b["events"].open_perf_buffer(cb)
try:
    while True: b.perf_buffer_poll()
except KeyboardInterrupt: pass
