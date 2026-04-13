# POC-023 — eBPF in Practice (without touching the host)

Hands-on **eBPF** lab using only Docker. Nothing is installed on the host
besides `docker` itself — the container runs `bpftrace` and shares the host
kernel (that's all eBPF really needs: a kernel, not packages).

---

## 1. What is eBPF (the minimum theory to not get lost)

**eBPF** (*extended Berkeley Packet Filter*) is a virtual machine that lives
inside the Linux kernel. You load a small program, the kernel **verifies**
that it is safe (no infinite loops, no invalid memory access, etc.) and then
**attaches** it to an execution point in the kernel — a *hook*. Whenever
that point executes, your program runs.

The points where you can attach eBPF programs are many:

| Hook | When it fires | Example use |
|---|---|---|
| **kprobe / kretprobe** | Entry/exit of any kernel function | Spy on `tcp_connect()` |
| **tracepoint** | Stable points defined by kernel maintainers | `sys_enter_openat` |
| **uprobe / uretprobe** | Entry/exit of userspace binary functions | Intercept OpenSSL `SSL_write` |
| **XDP** | Packet arriving on the NIC (before the TCP/IP stack) | High-speed firewall / DDoS |
| **tc (traffic control)** | Packet entering/leaving the qdisc | Shaping, observability |
| **LSM** | Kernel security hooks | AppArmor-style policies |
| **perf events** | CPU sampling, HW counters | Profiling |

**Why it's powerful:**

- You see what's happening **inside** the kernel, in production, without
  recompiling anything and without loading modules.
- The cost is very low — the program runs inline in the kernel, no context
  switch.
- It's **safe** by design: the *verifier* rejects programs that could crash
  the kernel.

**Tools you'll hear about:**

- **bcc** — Python + C framework for writing eBPF programs.
- **bpftrace** — a one-liner (awk-style) language for eBPF. That's what we
  use here because it's the fastest way to learn.
- **libbpf / CO-RE** — the "modern" way to write programs in C, portable
  across kernels. It's what projects like Cilium, Tetragon and Pixie use.

---

## 2. Why run it inside a container?

eBPF is a **kernel** feature. The container shares the host's kernel — so
when `bpftrace` inside the container loads a program, it sees **everything
happening on the host**: syscalls, packets, processes.

This means:

- You **don't** need to install `bpftrace`, `bcc-tools`, `clang`,
  `linux-headers` or anything on your laptop.
- You do need to give the container privileged capabilities (`SYS_ADMIN`,
  `BPF`, etc.) because loading eBPF programs is a kernel operation — this is
  expected.
- `pid: host` and `network_mode: host` make the container see real PIDs and
  real networking, so the outputs make sense.

The "dirty" side: the container has broad kernel access while it is running.
That's fine for a lab; in production you would restrict capabilities.

---

## 3. Prerequisites

- Linux with kernel **≥ 5.8** (check with `uname -r`) — any recent distro
  qualifies.
- Docker + `docker compose`.
- Running commands as root on the host is **not** required — it's the
  container that is privileged.

---

## 4. Bringing up the lab

Start the stack

```bash
docker compose up -d
```

This creates two containers:

| Container   | Role                                                     |
| ----------- | -------------------------------------------------------- |
| `ebpf-lab`  | `bpftrace` ready (`sleep infinity`), kernel hook host    |
| `ebpf-curl` | Traffic generator — `curl http://example.com` every 5s   |

Verify both containers are up

```bash
docker ps --filter name=ebpf --format 'table {{.Names}}\t{{.Status}}'
```

Expected output

```
NAMES       STATUS
ebpf-curl   Up 3 seconds
ebpf-lab    Up 3 seconds
```

Check the bpftrace version and the host kernel

```bash
docker exec ebpf-lab bpftrace --version
uname -r
```

Confirm that the kernel BTF is visible from inside the container (that's
what lets `bpftrace` resolve types like `struct sock`)

```bash
docker exec ebpf-lab ls -la /sys/kernel/btf/vmlinux
```

List a few tracepoints and kprobes to be sure the kernel exposes what the
scripts need

```bash
docker exec ebpf-lab bpftrace -l 'tracepoint:syscalls:sys_enter_openat'
docker exec ebpf-lab bpftrace -l 'kprobe:tcp_connect'
docker exec ebpf-lab bpftrace -l 'kprobe:tcp_sendmsg'
```

Open an interactive shell (optional — exercises can also run via
`docker exec`)

```bash
docker exec -it ebpf-lab bash
```

---

## 4.1 Testing and collecting evidence

Each script can be executed with `timeout` to run for a few seconds and dump
its output to a `.log` file, which serves as evidence of the experiment.

Create an evidence folder on the host

```bash
mkdir -p evidence
```

Test 1 — openat (normal volume of syscalls)

```bash
docker exec ebpf-lab timeout 3 bpftrace /scripts/01-hello.bt \
  > evidence/01-openat.log 2>&1
head -20 evidence/01-openat.log
```

Test 2 — execve (new processes on the host)

```bash
docker exec ebpf-lab timeout 5 bpftrace /scripts/02-execsnoop.bt \
  > evidence/02-execve.log 2>&1 &
# Meanwhile, trigger something so it shows up
docker exec ebpf-curl sh -c 'ls /tmp; echo hi'
wait
head -20 evidence/02-execve.log
```

Test 3 — TCP connections (generate traffic and capture in parallel)

```bash
docker exec ebpf-lab timeout 5 bpftrace /scripts/03-tcp-connect.bt \
  > evidence/03-tcp-connect.log 2>&1 &
sleep 1
docker exec ebpf-curl curl -s -o /dev/null http://example.com
wait
grep -E 'curl|example' evidence/03-tcp-connect.log || head -20 evidence/03-tcp-connect.log
```

Test 4 — packet capture per process

```bash
docker exec ebpf-lab timeout 5 bpftrace /scripts/04-packet-capture.bt \
  > evidence/04-packet.log 2>&1 &
sleep 1
docker exec ebpf-curl curl -s -o /dev/null http://example.com
wait
head -20 evidence/04-packet.log
```

Test 5 — aggregation with BPF maps (let it run and read the final report)

```bash
docker exec ebpf-lab timeout 8 bpftrace /scripts/05-bytes-per-process.bt \
  > evidence/05-bytes.log 2>&1
tail -30 evidence/05-bytes.log
```

At the end you should have

```bash
ls -la evidence/
# 01-openat.log
# 02-execve.log
# 03-tcp-connect.log
# 04-packet.log
# 05-bytes.log
```

Each file is the raw evidence of that experiment — real PIDs, comms, IPs and
byte counts from your host at the moment of execution. Useful for pasting in
tickets, posts, or just reviewing later.

### Troubleshooting quick reference

| Symptom                                               | Likely cause / fix                                              |
| ----------------------------------------------------- | --------------------------------------------------------------- |
| `ERROR: Could not resolve symbol: tcp_connect`        | kprobe renamed — run `bpftrace -l 'kprobe:tcp_*'` and adjust    |
| `ERROR: Kernel headers not found`                     | missing `/lib/modules` or `/usr/src` bind — already in compose  |
| No events appear                                      | kernel without BTF — check `/sys/kernel/btf/vmlinux`            |
| `Permission denied` loading BPF                       | container not running `--privileged` — recreate the stack      |

Tear down the lab when you're done

```bash
docker compose down
```

---

## 5. Hands-on — five exercises

The scripts live in [`scripts/`](scripts/). Run each one **inside** the
`ebpf-lab` container (`docker exec -it ebpf-lab bash`). Use Ctrl-C to stop.

### Exercise 1 — Hello, eBPF

```bash
bpftrace /scripts/01-hello.bt
```

Shows every process calling `openat(2)` on the host — the syscall used to
open files. You'll see a rain of events: this is the normal volume of kernel
activity, and eBPF can observe all of it without bringing anything down.

**What is happening**

- `tracepoint:syscalls:sys_enter_openat` is a *stable* kernel point.
- Every time any process (on the entire host, thanks to `pid: host`)
  executes that syscall, the program prints PID and command name.

### Exercise 2 — Snooping execs

```bash
bpftrace /scripts/02-execsnoop.bt
```

Now open another terminal and run `ls`, `cat /etc/hostname`, whatever. Each
new `execve()` shows up with a relative timestamp, PID and binary path.

This is the bpftrace version of `execsnoop` from `bcc-tools`. Useful to
audit what is being launched on a live machine in real time.

### Exercise 3 — Outbound TCP connections

```bash
bpftrace /scripts/03-tcp-connect.bt
```

In another shell, generate traffic:

```bash
docker exec ebpf-curl curl -s http://example.com -o /dev/null
```

You will see a line with PID, process name, destination IP and port. Note
that the program reads straight from the `struct sock *` the kernel passes
as an argument to `tcp_connect()` — that's what a *kprobe* gives you: the
ability to read kernel memory in real time without recompiling anything.

### Exercise 4 — "Capturing packets" per process

```bash
bpftrace /scripts/04-packet-capture.bt
```

Unlike `tcpdump`, which captures at the network device level (L2), this
script captures at the socket layer (`tcp_sendmsg`), so the **owning
process** of the packet is available. For each TCP send you see PID, comm,
destination, port and bytes.

This is one of eBPF's superpowers: correlating packet ↔ process without
contortions through `/proc/*/net`.

```bash
# Generate traffic while observing
docker exec ebpf-curl curl -s https://example.com -o /dev/null
```

### Exercise 5 — Aggregation with BPF maps

```bash
bpftrace /scripts/05-bytes-per-process.bt
```

Let it run for 30 seconds and press Ctrl-C. `bpftrace` prints the `@sent`
and `@recv` maps, showing how many bytes each process sent and received
over TCP during the period.

**The new concept here is the *BPF map*.** Maps are data structures shared
between the program running in the kernel and userspace. Without them you
could only print — with them you count, aggregate, build histograms, etc.

---

## 6. Experiments worth trying

- Swap `tcp_sendmsg` for `tcp_recvmsg` in script 04 and observe **return**
  traffic.
- Change script 01 to filter only `comm == "curl"`:
  ```
  tracepoint:syscalls:sys_enter_openat /comm == "curl"/ { ... }
  ```
- Run `bpftrace -l 'tracepoint:syscalls:*'` inside the container to list the
  thousands of tracepoints available.
- Try `bpftrace -l 'kprobe:tcp_*'` to see how many TCP kernel functions are
  exposed.

---

## 7. Tearing down the lab

```bash
docker compose down
```

Nothing gets installed on your machine. No traces, just what `docker` was
already using.

---

## 8. What to explore next (when you want to move past bpftrace)

1. **bcc** — when a script gets large, migrate to bcc (Python + C). Same
   hooks, more control.
2. **libbpf + CO-RE** — modern, production-grade approach. Compile once,
   run on many kernels thanks to BTF.
3. **Cilium / Tetragon / Falco / Pixie** — open source projects for
   networking, security and observability built on eBPF. Worth reading
   their BPF programs to see real-world patterns.

---

## 9. References

- [Brendan Gregg — eBPF book and tools](https://www.brendangregg.com/ebpf.html)
- [bpftrace reference guide](https://github.com/bpftrace/bpftrace/blob/master/docs/reference_guide.md)
- [ebpf.io](https://ebpf.io) — community hub
- [Learning eBPF — Liz Rice (O'Reilly)](https://isovalent.com/learning-ebpf/)
- [Linux kernel `bpf` docs](https://www.kernel.org/doc/html/latest/bpf/)

---

## 10. Files

```
023/
├── docker-compose.yml          — bpftrace container + traffic generator
├── README.md                   — This file
└── scripts/
    ├── 01-hello.bt             — Syscall tracepoint: openat
    ├── 02-execsnoop.bt         — Syscall tracepoint: execve
    ├── 03-tcp-connect.bt       — kprobe on tcp_connect
    ├── 04-packet-capture.bt    — kprobe on tcp_sendmsg (packet ↔ process)
    └── 05-bytes-per-process.bt — BPF maps + aggregation
```
