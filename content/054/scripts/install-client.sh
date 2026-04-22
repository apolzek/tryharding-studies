#!/usr/bin/env bash
# install-client.sh — build and install trynetd + trynet on a Linux host.
# Run as root. Assumes a Go 1.22+ toolchain and the WireGuard kernel module
# (or wireguard-go userspace implementation) available.

set -euo pipefail

if [[ $EUID -ne 0 ]]; then
  echo "must run as root" >&2
  exit 1
fi

CONTROL_URL="${1:-}"
RELAY_URL="${2:-}"

if [[ -z "$CONTROL_URL" ]]; then
  echo "usage: $0 <control-url> [relay-url]" >&2
  echo "example: $0 https://vpn.example.com:8443 wss://vpn.example.com:3478/relay" >&2
  exit 2
fi

cd "$(dirname "$0")/.."

echo "building binaries..."
go build -trimpath -o /usr/local/bin/trynetd ./cmd/trynetd
go build -trimpath -o /usr/local/bin/trynet  ./cmd/trynet

echo "writing /etc/trynet/config.json"
install -d -m 700 /etc/trynet /var/lib/trynet
cat > /etc/trynet/config.json <<EOF
{
  "control_url": "$CONTROL_URL",
  "relay_url":   "${RELAY_URL:-}",
  "insecure":    false,
  "listen_port": 41641
}
EOF
chmod 600 /etc/trynet/config.json

echo "installing systemd unit"
cat > /etc/systemd/system/trynetd.service <<'EOF'
[Unit]
Description=trynet client daemon
After=network-online.target
Wants=network-online.target

[Service]
ExecStart=/usr/local/bin/trynetd
Restart=always
RestartSec=3
AmbientCapabilities=CAP_NET_ADMIN CAP_NET_RAW
CapabilityBoundingSet=CAP_NET_ADMIN CAP_NET_RAW
ProtectSystem=strict
ReadWritePaths=/var/lib/trynet /run /etc/hosts
NoNewPrivileges=yes

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable --now trynetd.service

cat <<EOF

done.

next step — authenticate this machine:

  trynet up --authkey tskey-xxxxxxxxxx

then:

  trynet status
  trynet ip

EOF
