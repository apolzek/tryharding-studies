#!/usr/bin/env bash
# Setup de ambiente eBPF em Ubuntu 24.04.
#
# TUDO ABAIXO ESTÁ COMENTADO DE PROPÓSITO.
# Leia, decida o que quer, e descomente linha por linha.
# Nada é executado em um `bash install.sh` — é só referência.
#
# Rodar (depois de descomentar):   sudo bash 00-setup/install.sh
#
# -----------------------------------------------------------------------------

# if [[ $EUID -ne 0 ]]; then
#   echo "run with sudo"; exit 1
# fi

# ---- 1) pacotes do toolchain (clang/llvm + libbpf dev + headers) -----------
# sudo apt-get update
# sudo apt-get install -y --no-install-recommends \
#   clang llvm make \
#   libbpf-dev libelf-dev zlib1g-dev \
#   linux-tools-common "linux-tools-$(uname -r)" \
#   linux-headers-"$(uname -r)"

# ---- 2) BCC + bpftrace (já vêm pré-instalados no seu sistema) --------------
# sudo apt-get install -y bpfcc-tools python3-bpfcc bpftrace

# ---- 3) (opcional) bpftool mais novo, compilado do kernel source -----------
# sudo apt-get install -y git
# git clone --depth=1 https://github.com/libbpf/bpftool.git /tmp/bpftool
# make -C /tmp/bpftool/src
# sudo install -m 0755 /tmp/bpftool/src/bpftool /usr/local/sbin/bpftool

# ---- 4) checagens de sanidade ----------------------------------------------
# clang --version | head -1
# bpftool version | head -1
# bpftrace --version | head -1
# python3 -c "from bcc import BPF; print('bcc OK')"
# [[ -f /sys/kernel/btf/vmlinux ]] && echo "btf OK" || echo "btf MISSING"

# ---- 5) (opcional) aumentar memlock para programas BPF grandes -------------
# echo '* soft memlock unlimited' | sudo tee -a /etc/security/limits.conf
# echo '* hard memlock unlimited' | sudo tee -a /etc/security/limits.conf

# ---- 6) (opcional) permitir usuário comum carregar BPF não-privileged ------
# sudo sysctl -w kernel.unprivileged_bpf_disabled=0   # CUIDADO: só em lab

# ---- 7) montagens que os exemplos podem precisar ---------------------------
# sudo mount -t debugfs none /sys/kernel/debug    2>/dev/null || true
# sudo mount -t tracefs none /sys/kernel/tracing  2>/dev/null || true
