## eBPF hands-on lab with bpftrace in Docker

### Objectives

Provide a hands-on **eBPF** lab using only Docker — nothing is installed on the host besides `docker` itself. The container runs `bpftrace` and shares the host kernel (that is all eBPF really needs: a kernel, not packages). The goal is to observe syscalls, new processes, outbound TCP connections, per-process packet flow and BPF-map aggregation from inside a privileged container, building enough intuition with `bpftrace` one-liners to later graduate to `bcc` or `libbpf+CO-RE`.

eBPF (*extended Berkeley Packet Filter*) is a virtual machine inside the Linux kernel. You load a small program, the kernel verifies it is safe (no infinite loops, no invalid memory access), then attaches it to a hook — a kernel function, tracepoint, userspace binary, NIC, socket, etc. Whenever the hook fires, your program runs inline with very low overhead and full observability.

Common hooks:

| Hook | When it fires | Example use |
|---|---|---|
| kprobe / kretprobe | Entry/exit of any kernel function | Spy on `tcp_connect()` |
| tracepoint | Stable points defined by the kernel | `sys_enter_openat` |
| uprobe / uretprobe | Entry/exit of userspace binary functions | Intercept OpenSSL `SSL_write` |
| XDP | Packet arriving on the NIC (before TCP/IP stack) | High-speed firewall / DDoS |
| tc | Packet entering/leaving the qdisc | Shaping, observability |
| LSM | Kernel security hooks | AppArmor-style policies |
| perf events | CPU sampling, HW counters | Profiling |

### Prerequisites

- Linux with kernel ≥ 5.8 (check with `uname -r`)
- Docker + docker compose
- Running commands as root on the host is **not** required — it is the container that is privileged

### Reproducing

Bring up the lab:

```bash
docker compose up -d
```

This creates two containers:

| Container | Role |
|---|---|
| `ebpf-lab`  | `bpftrace` ready (`sleep infinity`), kernel hook host |
| `ebpf-curl` | Traffic generator — `curl http://example.com` every 5 s |

Verify containers and kernel support:

```bash
docker ps --filter name=ebpf --format 'table {{.Names}}\t{{.Status}}'
docker exec ebpf-lab bpftrace --version
docker exec ebpf-lab ls -la /sys/kernel/btf/vmlinux
docker exec ebpf-lab bpftrace -l 'tracepoint:syscalls:sys_enter_openat'
docker exec ebpf-lab bpftrace -l 'kprobe:tcp_connect'
```

Open an interactive shell (optional):

```bash
docker exec -it ebpf-lab bash
```

Run the five exercises (each under `scripts/`):

```bash
# 1 — Hello, eBPF: tracepoint on openat syscall
bpftrace /scripts/01-hello.bt

# 2 — execsnoop: every new exec on the host
bpftrace /scripts/02-execsnoop.bt

# 3 — Outbound TCP connections via kprobe on tcp_connect
bpftrace /scripts/03-tcp-connect.bt

# 4 — Packet ↔ process correlation via kprobe on tcp_sendmsg
bpftrace /scripts/04-packet-capture.bt

# 5 — Aggregation with BPF maps (bytes per process)
bpftrace /scripts/05-bytes-per-process.bt
```

Collect evidence automatically with `timeout` and redirect to log files:

```bash
mkdir -p evidence
docker exec ebpf-lab timeout 3 bpftrace /scripts/01-hello.bt > evidence/01-openat.log 2>&1
docker exec ebpf-lab timeout 5 bpftrace /scripts/03-tcp-connect.bt > evidence/03-tcp-connect.log 2>&1 &
sleep 1 && docker exec ebpf-curl curl -s -o /dev/null http://example.com && wait
docker exec ebpf-lab timeout 8 bpftrace /scripts/05-bytes-per-process.bt > evidence/05-bytes.log 2>&1
```

Troubleshooting:

| Symptom | Likely cause / fix |
|---|---|
| `Could not resolve symbol: tcp_connect` | kprobe renamed — run `bpftrace -l 'kprobe:tcp_*'` and adjust |
| `Kernel headers not found` | missing `/lib/modules` or `/usr/src` bind — already in compose |
| No events appear | kernel without BTF — check `/sys/kernel/btf/vmlinux` |
| `Permission denied` loading BPF | container not running `--privileged` — recreate the stack |

Tear down:

```bash
docker compose down
```

### Results

Running eBPF from inside a privileged container is a practical way to experiment without polluting the host: the container sees real PIDs, real networking and the real kernel, but uninstalling the lab is a single `docker compose down`. `bpftrace` is by far the fastest way to learn — kprobes give you immediate access to kernel function arguments (e.g. reading `struct sock *` inside `tcp_connect()`), and BPF maps turn print-only scripts into real aggregation tools. Once a one-liner grows past a handful of lines, the natural next step is `bcc` (Python + C) and then `libbpf + CO-RE` for production-grade, portable programs — the same foundation used by Cilium, Tetragon, Falco and Pixie.

### References

```
🔗 https://www.brendangregg.com/ebpf.html
🔗 https://github.com/bpftrace/bpftrace/blob/master/docs/reference_guide.md
🔗 https://ebpf.io
🔗 https://isovalent.com/learning-ebpf/
🔗 https://www.kernel.org/doc/html/latest/bpf/
```
