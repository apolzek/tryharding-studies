#!/usr/bin/env python3
"""
23 — Run queue length sampler: amostragem periódica da nr_running da runqueue.

perf_event sampler + helper bpf_get_smp_processor_id.

Run: sudo python3 23_runqlen.py 5     (amostra 5s e imprime)
Test: sudo make test-23
"""
import sys, time
from bcc import BPF, PerfType, PerfSWConfig

dur = int(sys.argv[1]) if len(sys.argv) > 1 else 5
prog = r"""
#include <linux/sched.h>
typedef struct { unsigned int val; } rq_slot;
BPF_HISTOGRAM(dist);

struct rq;
extern void *runqueues;

int sample(struct bpf_perf_event_data *ctx) {
    struct rq *rq;
    rq = (struct rq *)bpf_this_cpu_ptr(&runqueues);
    unsigned int n = 0;
    bpf_probe_read_kernel(&n, sizeof(n), rq);   /* nr_running is first int */
    dist.increment(bpf_log2l(n));
    return 0;
}
"""

b = BPF(text=prog)
b.attach_perf_event(ev_type=PerfType.SOFTWARE, ev_config=PerfSWConfig.CPU_CLOCK,
                    fn_name="sample", sample_period=0, sample_freq=99)
print(f"sampling for {dur}s...")
time.sleep(dur)
b["dist"].print_log2_hist("runqlen")
