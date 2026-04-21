# Makefile para exemplos libbpf (CO-RE).
# Requer: clang, llvm, libbpf-dev, libelf-dev, zlib1g-dev, bpftool.
#
# Target principal:  make          → compila tudo
# Rodar:             sudo ./minimal
# Limpar:            make clean

CLANG      ?= clang
BPFTOOL    ?= bpftool
ARCH       ?= $(shell uname -m | sed 's/x86_64/x86/' | sed 's/aarch64/arm64/')
VMLINUX_H  := vmlinux.h

CFLAGS_BPF := -g -O2 -target bpf -D__TARGET_ARCH_$(ARCH) -I. -I/usr/include/$(shell uname -m)-linux-gnu
CFLAGS     := -O2 -Wall -I.
LDLIBS     := -lbpf -lelf -lz

BPF_SRCS := minimal.bpf.c open_latency.bpf.c xdp_count.bpf.c
BPF_OBJS := $(BPF_SRCS:.bpf.c=.bpf.o)
SKELS    := $(BPF_SRCS:.bpf.c=.skel.h)

BINS := minimal open_latency

all: $(VMLINUX_H) $(BPF_OBJS) $(SKELS) $(BINS)

$(VMLINUX_H):
	$(BPFTOOL) btf dump file /sys/kernel/btf/vmlinux format c > $@

%.bpf.o: %.bpf.c $(VMLINUX_H)
	$(CLANG) $(CFLAGS_BPF) -c $< -o $@
	# opcional: strip para reduzir .BTF duplicado
	# llvm-strip -g $@

%.skel.h: %.bpf.o
	$(BPFTOOL) gen skeleton $< > $@

minimal: minimal.c minimal.skel.h
	$(CC) $(CFLAGS) $< -o $@ $(LDLIBS)

open_latency: open_latency.c open_latency.skel.h
	$(CC) $(CFLAGS) $< -o $@ $(LDLIBS)

clean:
	rm -f $(BPF_OBJS) $(SKELS) $(BINS) $(VMLINUX_H)

.PHONY: all clean
