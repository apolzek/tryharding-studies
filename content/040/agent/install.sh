#!/usr/bin/env bash
# Install the userwatch eBPF agent on Ubuntu 22.04+/24.04.
set -euo pipefail

if [[ $EUID -ne 0 ]]; then
  echo "Run as root: sudo $0" >&2
  exit 1
fi

DEST=/opt/userwatch/agent
SRC="$(cd "$(dirname "$0")" && pwd)"

echo "==> Installing system packages"
apt-get update -y
apt-get install -y --no-install-recommends \
  bpfcc-tools \
  python3-bpfcc \
  python3 \
  linux-headers-generic \
  "linux-headers-$(uname -r)" || true

echo "==> Copying agent to $DEST"
install -d -m 0755 "$DEST"
install -m 0644 "$SRC/bpf_program.c" "$DEST/"
install -m 0755 "$SRC/agent.py" "$DEST/"

echo "==> Installing systemd unit"
install -m 0644 "$SRC/userwatch-agent.service" /etc/systemd/system/

systemctl daemon-reload
echo
echo "Done. Enable and start with:"
echo "  sudo systemctl enable --now userwatch-agent"
echo "  sudo journalctl -u userwatch-agent -f"
