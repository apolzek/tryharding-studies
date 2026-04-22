# trynet

A self-hosted, Tailscale-style mesh VPN written in Go. One binary runs on your
VPS (control plane + relay + admin UI), one binary runs on every device, and
the nodes peer with each other over WireGuard.

**Not affiliated with Tailscale.** This is a study / POC; see
[`docs/FEATURES.md`](docs/FEATURES.md) for the feature-by-feature gap list.

## Quick look

- **Control server** (`trynet-control`) — HTTP API, netmap distribution, ACL engine.
- **Relay** (`trynet-derp`) — WebSocket relay for NAT'd peers.
- **Admin UI** (`trynet-ui`) — web console.
- **Daemon** (`trynetd`) — runs on each device, drives WireGuard via `wgctrl`.
- **CLI** (`trynet`) — talks to the local daemon.

```
.
├── cmd/
│   ├── trynet/            # CLI
│   ├── trynetd/           # client daemon
│   ├── trynet-control/    # coordination server
│   ├── trynet-derp/       # relay
│   └── trynet-ui/         # admin UI
├── internal/
│   ├── protocol/          # shared types (Node, NetMap, ACL, ...)
│   ├── crypto/            # key generation helpers
│   ├── control/           # control-plane logic
│   ├── derp/              # relay server
│   ├── client/            # daemon internals (wg, dns, netmap loop)
│   ├── cli/               # CLI commands
│   └── ui/                # admin templates + HTTP handlers
├── docs/                  # FEATURES + ARCHITECTURE
├── scripts/               # install scripts
├── docker-compose.yml     # VPS services
└── go.mod
```

## VPS setup (one box)

```bash
# on the VPS
git clone <this repo> /opt/trynet && cd /opt/trynet/content/054
cp scripts/env.example .env           # edit TRYNET_ADMIN_TOKEN, domain
docker compose up -d --build
```

Expose:
- `TCP 8443` — control API
- `TCP 8080` — admin UI
- `TCP 3478` — relay (WebSocket)

## Client setup

```bash
# on a Linux box (needs root for wireguard)
go build -o /usr/local/bin/trynetd ./cmd/trynetd
go build -o /usr/local/bin/trynet  ./cmd/trynet

sudo mkdir -p /etc/trynet
cat <<EOF | sudo tee /etc/trynet/config.json
{
  "control_url": "https://vpn.example.com:8443",
  "relay_url":   "wss://vpn.example.com:3478",
  "insecure":    false
}
EOF

sudo systemctl start trynetd   # or run directly for debugging
sudo trynet up --authkey tskey-xxxxxxxxxxxx
trynet status
```

See [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) for the deeper design and
[`docs/FEATURES.md`](docs/FEATURES.md) for what's implemented vs. left-as-stub.
