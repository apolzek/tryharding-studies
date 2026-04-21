// 52 — openat latency histogram em C puro (CO-RE).
#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>

char LICENSE[] SEC("license") = "GPL";

struct { __uint(type, BPF_MAP_TYPE_HASH);
         __type(key, __u32); __type(value, __u64);
         __uint(max_entries, 8192); } start SEC(".maps");

struct { __uint(type, BPF_MAP_TYPE_ARRAY);
         __type(key, __u32); __type(value, __u64);
         __uint(max_entries, 32); } dist SEC(".maps");

SEC("tracepoint/syscalls/sys_enter_openat")
int enter(void *ctx) {
    __u32 tid = bpf_get_current_pid_tgid();
    __u64 t = bpf_ktime_get_ns();
    bpf_map_update_elem(&start, &tid, &t, 0);
    return 0;
}

SEC("tracepoint/syscalls/sys_exit_openat")
int exit(void *ctx) {
    __u32 tid = bpf_get_current_pid_tgid();
    __u64 *t = bpf_map_lookup_elem(&start, &tid);
    if (!t) return 0;
    __u64 dt_us = (bpf_ktime_get_ns() - *t) / 1000;
    bpf_map_delete_elem(&start, &tid);

    __u32 slot = 0;
    for (__u64 x = dt_us; x >>= 1; slot++) {}
    if (slot >= 32) slot = 31;
    __u64 *v = bpf_map_lookup_elem(&dist, &slot);
    if (v) __sync_fetch_and_add(v, 1);
    return 0;
}
