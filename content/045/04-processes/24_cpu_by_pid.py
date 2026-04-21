#!/usr/bin/env python3
"""
24 — CPU profile: amostra on-CPU a 99Hz, conta por PID/comm.

Run: sudo python3 24_cpu_by_pid.py 5   (perfil de 5s)
Test: sudo make test-24
"""
import sys, time
from bcc import BPF, PerfType, PerfSWConfig

dur = int(sys.argv[1]) if len(sys.argv) > 1 else 5
prog = r"""
struct key_t { u32 pid; char comm[16]; };
BPF_HASH(cnt, struct key_t, u64);

int sample(struct bpf_perf_event_data *ctx) {
    struct key_t k = {};
    k.pid = bpf_get_current_pid_tgid() >> 32;
    bpf_get_current_comm(&k.comm, sizeof(k.comm));
    u64 z = 0, *v = cnt.lookup_or_try_init(&k, &z);
    if (v) __sync_fetch_and_add(v, 1);
    return 0;
}
"""

b = BPF(text=prog)
b.attach_perf_event(ev_type=PerfType.SOFTWARE, ev_config=PerfSWConfig.CPU_CLOCK,
                    fn_name="sample", sample_period=0, sample_freq=99)
print(f"profiling for {dur}s...")
time.sleep(dur)
rows = [(k.pid, k.comm.decode('utf-8','replace'), v.value)
        for k, v in b["cnt"].items()]
rows.sort(key=lambda r: -r[2])
print(f"\n{'PID':<8}{'COMM':<20}{'SAMPLES':>10}")
for p, c, n in rows[:25]:
    print(f"{p:<8}{c:<20}{n:>10}")
