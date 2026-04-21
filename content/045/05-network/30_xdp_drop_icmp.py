#!/usr/bin/env python3
"""
30 — XDP: derruba todo ICMP entrando na interface.

O programa roda *antes* da pilha de rede — performance máxima.
USA MODO GENERIC por segurança: funciona em qualquer driver (incl. lo).
PARA TESTE: usa interface 'lo' por padrão (ping 127.0.0.1 vai parar).

Run:     sudo python3 30_xdp_drop_icmp.py [iface]
Test: sudo make test-30
Stop:    Ctrl-C (remove o hook)
"""
import sys, time
from bcc import BPF

iface = sys.argv[1] if len(sys.argv) > 1 else "lo"
prog = r"""
#include <linux/bpf.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <linux/in.h>

int xdp_drop_icmp(struct xdp_md *ctx) {
    void *data     = (void *)(long)ctx->data;
    void *data_end = (void *)(long)ctx->data_end;

    struct ethhdr *eth = data;
    if ((void *)(eth + 1) > data_end) return XDP_PASS;
    if (eth->h_proto != bpf_htons(ETH_P_IP)) return XDP_PASS;

    struct iphdr *ip = (void *)(eth + 1);
    if ((void *)(ip + 1) > data_end) return XDP_PASS;
    if (ip->protocol == IPPROTO_ICMP) return XDP_DROP;
    return XDP_PASS;
}
"""

b = BPF(text=prog)
fn = b.load_func("xdp_drop_icmp", BPF.XDP)
# modo SKB/GENERIC é compatível com 'lo' e todos drivers
b.attach_xdp(iface, fn, flags=(1 << 1))  # XDP_FLAGS_SKB_MODE

print(f"dropping ICMP on {iface}. Ctrl-C to detach.")
try:
    while True: time.sleep(1)
except KeyboardInterrupt: pass
finally:
    b.remove_xdp(iface, flags=(1 << 1))
    print("detached.")
