// 53 — XDP em C puro: conta pacotes por protocolo L4.
#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>

char LICENSE[] SEC("license") = "GPL";

struct { __uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
         __type(key, __u32); __type(value, __u64);
         __uint(max_entries, 4); } cnt SEC(".maps");

SEC("xdp")
int xdp_count(struct xdp_md *ctx) {
    void *data = (void *)(long)ctx->data;
    void *end  = (void *)(long)ctx->data_end;
    struct ethhdr *e = data;
    if ((void *)(e + 1) > end) return XDP_PASS;
    if (e->h_proto != bpf_htons(ETH_P_IP)) return XDP_PASS;
    struct iphdr *ip = (void *)(e + 1);
    if ((void *)(ip + 1) > end) return XDP_PASS;

    __u32 k = 3;
    if (ip->protocol == IPPROTO_TCP) k = 0;
    else if (ip->protocol == IPPROTO_UDP) k = 1;
    else if (ip->protocol == IPPROTO_ICMP) k = 2;
    __u64 *v = bpf_map_lookup_elem(&cnt, &k);
    if (v) (*v)++;
    return XDP_PASS;
}
