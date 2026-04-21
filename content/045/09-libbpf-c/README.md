# libbpf + CO-RE (C puro)

Exemplos sem BCC. Compilam para um objeto BPF (`.bpf.o`) + skeleton
gerado pelo `bpftool` + binário userspace linkado com `libbpf`.

## Requisitos

```bash
sudo apt install -y clang llvm libbpf-dev libelf-dev zlib1g-dev bpftool
```

## Build

```bash
make
```

O Makefile:
1. Gera `vmlinux.h` a partir do BTF do kernel (`/sys/kernel/btf/vmlinux`).
2. Compila cada `.bpf.c` → `.bpf.o` (bytecode BPF).
3. Gera `*.skel.h` com `bpftool gen skeleton`.
4. Compila e linka o loader em C.

## Run

```bash
sudo ./minimal          # 51 — exec snoop via ring buffer
sudo ./open_latency     # 52 — histograma de openat() latency
# 53 — xdp_count.bpf.o é anexável via:
sudo bpftool prog load xdp_count.bpf.o /sys/fs/bpf/xdp_count
sudo bpftool net attach xdp pinned /sys/fs/bpf/xdp_count dev lo
sudo bpftool map dump name cnt
sudo bpftool net detach xdp dev lo
sudo rm /sys/fs/bpf/xdp_count
```

## O que você aprende aqui

- **CO-RE**: um binário rodando em kernels diferentes (graças ao BTF +
  relocação de campos). BCC recompila no alvo; libbpf não.
- **Skeleton**: API gerada estaticamente (tipos C fortes para maps/progs,
  sem ioctls manuais).
- **ring buffer** em C puro (`bpf_ringbuf_reserve/submit`).
- **XDP standalone**: objeto que o `bpftool` anexa sem userspace rodando.
