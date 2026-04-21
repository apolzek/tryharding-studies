# eBPF cheatsheet

## BCC helpers mais usados

| Helper | O que faz |
|---|---|
| `bpf_get_current_pid_tgid()` | `(tgid<<32) \| pid` — pid = TID, tgid = PID |
| `bpf_get_current_uid_gid()` | `(gid<<32) \| uid` |
| `bpf_get_current_comm(&c, sizeof(c))` | nome do processo (TASK_COMM_LEN=16) |
| `bpf_ktime_get_ns()` | ns desde boot (monotonic) |
| `bpf_probe_read_user(dst, sz, src)` | copia memória de userspace |
| `bpf_probe_read_kernel(dst, sz, src)` | copia memória do kernel |
| `bpf_trace_printk("fmt", ...)` | escreve em trace_pipe (debug only) |
| `events.perf_submit(ctx, &data, sizeof(data))` | envia evento ao userspace |

## Tipos de map (BCC)

- `BPF_HASH(name, key_t, val_t)` — mapa genérico
- `BPF_ARRAY(name, val_t, n)` — array de tamanho fixo
- `BPF_PERCPU_ARRAY(...)` — per-CPU, sem contenção
- `BPF_HISTOGRAM(name)` — p/ histogramas (log2)
- `BPF_PERF_OUTPUT(events)` — eventos → userspace
- `BPF_RINGBUF_OUTPUT(events, n_pages)` — kernel ≥ 5.8, mais eficiente

## Hooks mais comuns

| Hook | Quando dispara |
|---|---|
| `kprobe:<fn>` | entrada de função do kernel |
| `kretprobe:<fn>` | retorno da função |
| `tracepoint:<grupo>:<nome>` | tracepoint estático (mais estável) |
| `uprobe:/bin/bash:readline` | entrada de função em userspace |
| `xdp` | pacote chegando na NIC (antes da pilha) |
| `tc` | ingress/egress da queueing discipline |
| `perf_event` | amostragem periódica (profiling) |

## Atalhos de `bpftool`

```bash
sudo bpftool prog          # lista programas
sudo bpftool map           # lista maps
sudo bpftool map dump id N # dump valores de um map
sudo bpftool net           # XDP/tc anexados
sudo bpftool cgroup tree   # progs BPF em cgroups
```
