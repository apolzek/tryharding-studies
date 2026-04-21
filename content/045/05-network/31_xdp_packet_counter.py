#!/usr/bin/env python3
"""
31 — XDP per-protocol packet counter (TCP/UDP/ICMP/other).

BPF_PERCPU_ARRAY evita contenção entre CPUs.

Run: sudo python3 31_xdp_packet_counter.py [iface]
Test: sudo make test-31
"""
import sys, time
from bcc import BPF

iface = sys.argv[1] if len(sys.argv) > 1 else "lo"
prog = r"""
#include <linux/bpf.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <linux/in.h>

BPF_PERCPU_ARRAY(cnt, long, 4);   /* 0=TCP 1=UDP 2=ICMP 3=OTHER */

int xdp_count(struct xdp_md *ctx) {
    void *data = (void *)(long)ctx->data, *end = (void *)(long)ctx->data_end;
    struct ethhdr *e = data;
    if ((void *)(e+1) > end) return XDP_PASS;
    if (e->h_proto != bpf_htons(ETH_P_IP)) return XDP_PASS;
    struct iphdr *ip = (void *)(e+1);
    if ((void *)(ip+1) > end) return XDP_PASS;
    u32 key = 3;
    if (ip->protocol == IPPROTO_TCP)  key = 0;
    else if (ip->protocol == IPPROTO_UDP) key = 1;
    else if (ip->protocol == IPPROTO_ICMP) key = 2;
    long *v = cnt.lookup(&key);
    if (v) *v += 1;
    return XDP_PASS;
}
"""

b = BPF(text=prog)
fn = b.load_func("xdp_count", BPF.XDP)
b.attach_xdp(iface, fn, flags=(1 << 1))

try:
    names = ["TCP", "UDP", "ICMP", "OTHER"]
    while True:
        time.sleep(2)
        row = {n: sum(v.value for v in b["cnt"][i]) for i, n in enumerate(names)}
        print(f"TCP={row['TCP']:<8} UDP={row['UDP']:<8} "
              f"ICMP={row['ICMP']:<6} OTHER={row['OTHER']}")
except KeyboardInterrupt: pass
finally:
    b.remove_xdp(iface, flags=(1 << 1))
