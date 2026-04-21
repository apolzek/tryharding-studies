// 52 — loader para open_latency.bpf.c.
#include <stdio.h>
#include <unistd.h>
#include <signal.h>
#include <bpf/libbpf.h>
#include "open_latency.skel.h"

static volatile int stop;
static void on_sigint(int _) { stop = 1; }

int main(void) {
    struct open_latency_bpf *skel = open_latency_bpf__open_and_load();
    if (!skel) return 1;
    if (open_latency_bpf__attach(skel)) return 1;

    signal(SIGINT, on_sigint);
    puts("measuring openat() latency; Ctrl-C to print histogram");
    while (!stop) sleep(1);

    int fd = bpf_map__fd(skel->maps.dist);
    printf("\n  slot  |  count\n  ------+---------\n");
    for (__u32 i = 0; i < 32; i++) {
        __u64 v = 0;
        if (bpf_map_lookup_elem(fd, &i, &v) == 0 && v) {
            printf("  %5u | %8llu  (%llu..%llu us)\n",
                   i, v, (1ULL << i), (1ULL << (i+1)) - 1);
        }
    }
    open_latency_bpf__destroy(skel);
    return 0;
}
