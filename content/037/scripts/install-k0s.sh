#!/usr/bin/env bash
set -euo pipefail

: "${K0S_VERSION:?K0S_VERSION is required}"

arch="$(uname -m)"
case "$arch" in
  x86_64)  k0s_arch="amd64"  ;;
  aarch64) k0s_arch="arm64"  ;;
  *) echo "unsupported arch: $arch" >&2; exit 1 ;;
esac

url="https://github.com/k0sproject/k0s/releases/download/${K0S_VERSION}/k0s-${K0S_VERSION}-${k0s_arch}"

curl -fsSL --retry 5 --retry-delay 2 -o /usr/local/bin/k0s "$url"
chmod 0755 /usr/local/bin/k0s

install -d -m 0700 /etc/k0s /var/lib/k0s
ln -sf /usr/local/bin/k0s /usr/local/bin/kubectl
