# Makefile de testes — content/045/
#
# Precisa sudo (eBPF). Rode sempre com: sudo make <target>
#
# Principais:
#   sudo make help
#   sudo make test-01         # testa o exemplo 01 sozinho
#   sudo make test-basics     # testa todos os 01-basics/
#   sudo make test-bcc        # todos os Python/BCC
#   sudo make test-bpftrace   # todos os .bt
#   sudo make test-libbpf     # build + test dos C+libbpf
#   sudo make test-all        # tudo (não-destrutivo). ~5-10 min.
#   sudo make test-destructive  # XDP drop ICMP (cuidado em rede real)
#   make status               # imprime STATUS.md
#   make clean                # limpa .test-logs/ e STATUS.md

SHELL     := bash
TIMEOUT   ?= 6
TESTSH    := ./tools/test-one.sh
LOGDIR    := .test-logs
STATUS    := STATUS.md

export TIMEOUT LOGDIR STATUS

# ---------- inventário ------------------------------------------------------
BCC_FILES := \
  01-basics/01_hello.py \
  01-basics/02_syscall_counter.py \
  01-basics/03_perf_event.py \
  01-basics/04_hash_map.py \
  01-basics/05_histogram.py \
  02-syscalls/06_execve_snoop.py \
  02-syscalls/07_openat_snoop.py \
  02-syscalls/08_tcp_connect.py \
  02-syscalls/09_accept_snoop.py \
  02-syscalls/10_kill_snoop.py \
  02-syscalls/11_unlink_snoop.py \
  02-syscalls/12_syscall_count_by_comm.py \
  03-file-io/13_file_open_latency.py \
  03-file-io/14_rw_bytes_by_pid.py \
  03-file-io/15_fd_usage.py \
  03-file-io/16_slow_vfs_read.py \
  03-file-io/17_fs_ops_by_fstype.py \
  03-file-io/18_dir_listings.py \
  04-processes/19_execsnoop_args.py \
  04-processes/20_exit_snoop.py \
  04-processes/21_fork_tree.py \
  04-processes/22_oom_kill.py \
  04-processes/23_runqlen.py \
  04-processes/24_cpu_by_pid.py \
  05-network/25_tcp_lifecycle.py \
  05-network/26_tcp_retransmit.py \
  05-network/27_tcp_rtt_hist.py \
  05-network/28_udp_snoop.py \
  05-network/29_dns_qname.py \
  05-network/31_xdp_packet_counter.py \
  05-network/32_tcp_send_bytes_by_pid.py \
  06-performance/33_biolatency.py \
  06-performance/34_biosnoop.py \
  06-performance/35_runqlat.py \
  06-performance/36_offcputime_by_pid.py \
  06-performance/37_cachestat.py \
  06-performance/38_softirqs.py \
  07-security/39_capable.py \
  07-security/40_setuid_snoop.py \
  07-security/41_mount_snoop.py \
  07-security/42_ptrace_snoop.py \
  07-security/43_module_load.py \
  07-security/44_denied_open.py \
  08-advanced/45_bash_readline.py \
  08-advanced/46_ssl_snoop.py \
  08-advanced/49_stack_traces.py \
  08-advanced/50_ringbuf_events.py

# Precisa de argumento → testa com alvo estável
BCC_WITH_ARGS := 08-advanced/48_latency_hist_per_fn.py

# Pulados em test-all (destrutivos ou exigem PID externo)
BCC_SKIP := \
  05-network/30_xdp_drop_icmp.py \
  08-advanced/47_python_calls.py

BT_FILES := $(wildcard 10-bpftrace/*.bt)

# C+libbpf binaries
LIBBPF_DIR  := 09-libbpf-c
LIBBPF_BINS := $(LIBBPF_DIR)/minimal $(LIBBPF_DIR)/open_latency

# ---------- help ------------------------------------------------------------
.PHONY: help
help:
	@echo "make test-NN           # NN = 01..61 (um exemplo)"
	@echo "make test-basics       # pasta 01-basics/"
	@echo "make test-syscalls     # pasta 02-syscalls/"
	@echo "make test-file-io"
	@echo "make test-processes"
	@echo "make test-network"
	@echo "make test-performance"
	@echo "make test-security"
	@echo "make test-advanced"
	@echo "make test-bcc          # todos os .py"
	@echo "make test-bpftrace     # todos os .bt"
	@echo "make test-libbpf       # build + run dos C+libbpf"
	@echo "make test-all          # TUDO não-destrutivo"
	@echo "make test-destructive  # XDP drop ICMP (cuidado!)"
	@echo "make build-libbpf      # apenas build"
	@echo "make status            # mostra STATUS.md"
	@echo "make clean             # limpa logs"
	@echo ""
	@echo "Precisa sudo. TIMEOUT=$(TIMEOUT)s por exemplo (customize: TIMEOUT=10 sudo make ...)"

# ---------- utilitários ------------------------------------------------------
$(LOGDIR):
	@mkdir -p $(LOGDIR)

.PHONY: status clean
status:
	@[[ -f $(STATUS) ]] && cat $(STATUS) || echo "no status yet — run a test first"

clean:
	@rm -rf $(LOGDIR) $(STATUS)
	@echo "cleaned."

# ---------- test-NN: descoberta dinâmica pelo número -----------------------
# Ex.: make test-23 → acha 04-processes/23_runqlen.py
.PHONY: test-%
test-%: | $(LOGDIR)
	@f=$$(find . -maxdepth 3 \( -name "$*_*.py" -o -name "$*_*.bt" \) \
	        -not -path "./$(LOGDIR)/*" -not -path "./tools/*" | head -1); \
	if [[ -z "$$f" ]]; then \
	  echo "no example with prefix '$*' found"; exit 1; \
	fi; \
	$(TESTSH) "$${f#./}"

# ---------- categorias (pastas) --------------------------------------------
.PHONY: test-basics test-syscalls test-file-io test-processes \
        test-network test-performance test-security test-advanced

test-basics: | $(LOGDIR)
	@for f in $(filter 01-basics/%,$(BCC_FILES)); do $(TESTSH) "$$f" || true; done

test-syscalls: | $(LOGDIR)
	@for f in $(filter 02-syscalls/%,$(BCC_FILES)); do $(TESTSH) "$$f" || true; done

test-file-io: | $(LOGDIR)
	@for f in $(filter 03-file-io/%,$(BCC_FILES)); do $(TESTSH) "$$f" || true; done

test-processes: | $(LOGDIR)
	@for f in $(filter 04-processes/%,$(BCC_FILES)); do $(TESTSH) "$$f" || true; done

test-network: | $(LOGDIR)
	@for f in $(filter 05-network/%,$(BCC_FILES)); do $(TESTSH) "$$f" || true; done

test-performance: | $(LOGDIR)
	@for f in $(filter 06-performance/%,$(BCC_FILES)); do $(TESTSH) "$$f" || true; done

test-security: | $(LOGDIR)
	@for f in $(filter 07-security/%,$(BCC_FILES)); do $(TESTSH) "$$f" || true; done

test-advanced: | $(LOGDIR)
	@for f in $(filter 08-advanced/%,$(BCC_FILES)); do $(TESTSH) "$$f" || true; done
	@$(TESTSH) 08-advanced/48_latency_hist_per_fn.py vfs_read || true

# ---------- super-agregadores ----------------------------------------------
.PHONY: test-bcc test-bpftrace test-libbpf test-all test-destructive

test-bcc: | $(LOGDIR)
	@echo "== BCC (Python) =="
	@for f in $(BCC_FILES); do $(TESTSH) "$$f" || true; done
	@$(TESTSH) 08-advanced/48_latency_hist_per_fn.py vfs_read || true

test-bpftrace: | $(LOGDIR)
	@echo "== bpftrace =="
	@for f in $(BT_FILES); do $(TESTSH) "$$f" || true; done

test-libbpf: build-libbpf | $(LOGDIR)
	@echo "== libbpf-c =="
	@for bin in $(LIBBPF_BINS); do \
	   [[ -x "$$bin" ]] && $(TESTSH) "$$bin" || echo "skip $$bin (not built)"; \
	 done

test-all: | $(LOGDIR)
	@echo "# STATUS — $$(date -Is)" > $(STATUS)
	@$(MAKE) --no-print-directory test-bcc
	@$(MAKE) --no-print-directory test-bpftrace
	@$(MAKE) --no-print-directory test-libbpf || true
	@echo ""
	@echo "-- resumo --"
	@pass=$$(grep -c '^- \[x\]' $(STATUS) 2>/dev/null || echo 0); \
	 fail=$$(grep -c '^- \[ \]' $(STATUS) 2>/dev/null || echo 0); \
	 echo "PASS: $$pass   FAIL: $$fail   (detalhes em $(STATUS))"

test-destructive: | $(LOGDIR)
	@echo "!! drops ICMP em lo por $(TIMEOUT)s (ping 127.0.0.1 vai falhar) !!"
	@$(TESTSH) 05-network/30_xdp_drop_icmp.py lo || true

# ---------- libbpf C build --------------------------------------------------
.PHONY: build-libbpf
build-libbpf:
	@$(MAKE) -C $(LIBBPF_DIR)
