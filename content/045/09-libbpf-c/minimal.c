// 51 — loader/userspace do minimal.bpf.c usando libbpf + skeleton.
// Build: make  (gera vmlinux.h, skeleton, e linka libbpf)
#include <stdio.h>
#include <signal.h>
#include <unistd.h>
#include <bpf/libbpf.h>
#include "minimal.skel.h"

struct event { __u32 pid; char comm[16]; };

static volatile int stop;
static void on_sigint(int _) { stop = 1; }

static int on_event(void *ctx, void *data, size_t size) {
    const struct event *e = data;
    printf("%-8u %s\n", e->pid, e->comm);
    return 0;
}

int main(void) {
    struct minimal_bpf *skel = minimal_bpf__open_and_load();
    if (!skel) { fprintf(stderr, "open_and_load failed\n"); return 1; }
    if (minimal_bpf__attach(skel)) { fprintf(stderr, "attach failed\n"); return 1; }

    struct ring_buffer *rb = ring_buffer__new(bpf_map__fd(skel->maps.events),
                                              on_event, NULL, NULL);
    if (!rb) { fprintf(stderr, "ringbuf new failed\n"); return 1; }

    signal(SIGINT, on_sigint);
    printf("%-8s %s\n", "PID", "COMM");
    while (!stop) ring_buffer__poll(rb, 200);

    ring_buffer__free(rb);
    minimal_bpf__destroy(skel);
    return 0;
}
