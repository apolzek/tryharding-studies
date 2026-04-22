# trynet architecture

```
                                   ┌────────────────────────────────┐
                                   │               VPS              │
                                   │                                │
                                   │  ┌──────────┐    ┌──────────┐  │
           HTTPS (register,        │  │ control  │◀──▶│   UI     │  │
           poll-netmap, endpoints) │  │  :8443   │    │  :8080   │  │
   ┌───────────────────────────────┼─▶│          │    │          │  │
   │                               │  │  JSON    │    └──────────┘  │
   │                               │  │  store   │                  │
   │           WebSocket (relay)   │  └──────────┘    ┌──────────┐  │
   │    ┌──────────────────────────┼──────────────────│   derp   │  │
   │    │                          │                  │  :3478   │  │
   │    │                          │                  └──────────┘  │
   │    │                          └────────────────────────────────┘
   │    │
   │    │   direct WireGuard                ┌──────────────────┐
   │    │   (UDP) whenever reachable        │                  │
┌──▼────▼───────────┐      ◀──────────▶     │   trynetd        │
│    trynetd        │                       │   (peer B)       │
│    (peer A)       │                       │                  │
│                   │                       │   wg0            │
│   wg0             │                       │   100.64.0.3     │
│   100.64.0.2      │                       │                  │
│                   │                       │   /etc/hosts     │
│   /etc/hosts      │                       │   (MagicDNS)     │
│   (MagicDNS)      │                       │                  │
└───────────────────┘                       └──────────────────┘
         ▲
         │ UNIX socket (/run/trynetd.sock)
         │
  ┌──────┴──────┐
  │   trynet    │   CLI: up / down / status / ip / authkey
  └─────────────┘
```

## Components

### `trynet-control` (VPS)

- HTTP API on `:8443` (TLS).
- Endpoints:
  - `POST /machine/register` — enroll with a pre-auth key, return node key + tailnet IP.
  - `GET  /machine/map` — long-poll for netmap updates (30 s keep-alive).
  - `POST /machine/endpoints` — push discovered UDP endpoints (self-reported STUN).
  - `POST /machine/logout` — explicit disconnect.
  - `GET  /admin/*` — JSON admin API used by the UI.
- Persists to `state.json` (atomic rename on write).
- Pushes netmap over an in-memory fan-out bus — each connected node has a
  goroutine reading from a channel the `control.Hub` writes to.

### `trynet-derp` (VPS)

- WebSocket relay on `:3478/tls`.
- Each connecting client identifies itself with its node public key.
- Client sends frames `{dst_key, payload}`; server forwards to the matching
  connection if present.
- Ships a health endpoint `GET /healthz`.

### `trynet-ui` (VPS)

- Server-rendered admin console on `:8080`.
- Talks to the control server's `/admin/*` API with a shared secret.
- Pages: `/`, `/nodes`, `/keys`, `/acls`, `/settings`, `/logs`.
- Uses only `html/template` and plain CSS — no JS bundler.

### `trynetd` (client daemon)

- Long-running daemon, one per node.
- Reads `/etc/trynet/config.json` for control URL + auth key on first boot.
- Lifecycle:
  1. **Register** with control using a generated WireGuard keypair.
  2. Receive tailnet IP + initial netmap.
  3. Configure `wg0` interface via `wgctrl`.
  4. Rewrite `/etc/hosts` with the MagicDNS block.
  5. Open a long-poll for netmap updates.
  6. Periodically POST discovered endpoints.
  7. Expose UNIX socket for the CLI and HTTP Taildrop endpoint on the tailnet IP.
- If WireGuard direct fails, the `wg0` interface points at the DERP-relayed
  virtual endpoint for that peer.

### `trynet` (CLI)

- Talks to `trynetd` over `/run/trynetd.sock`.
- Does no WireGuard work itself — delegates.

## Data model

See `internal/protocol/types.go`. The key types are:

- `Node`         — machine, owns a node key, a tailnet IP, tags, endpoints.
- `PreAuthKey`   — token used to enroll a node; can be reusable/ephemeral.
- `ACLPolicy`    — parsed HuJSON-like policy (src/dst/proto/ports).
- `NetMap`       — what a single node sees: self + peers + DNS + derp map + routes.

## Netmap distribution

1. Any mutation on the control server (`AddNode`, `SetACL`, `ExpireKey`) bumps
   an atomic `tailnet.version`.
2. The `Hub` walks each subscriber, computes their *personalized* netmap
   (ACLs filter peers down to what that node is allowed to see), and writes
   it to the subscriber's channel.
3. The client's long-poll returns with the new netmap; the agent diffs and
   applies.

## Security notes

- Control uses TLS with a cert mounted into the container (Let's Encrypt or
  self-signed for dev).
- Pre-auth keys are 32-byte URL-safe tokens, single-use or reusable.
- Node keys are Curve25519 pairs generated client-side; the private key never
  leaves the node.
- The relay never sees plaintext — it shuttles already-encrypted WireGuard
  frames addressed by public key.
- Admin API is protected by a shared secret passed in `X-Admin-Token`.

## Out-of-band decisions worth knowing

- IPv4 CGNAT range `100.64.0.0/10` is used as the tailnet address space — same
  choice as Tailscale. IPs are allocated sequentially with a bitmap held by
  the control server.
- Default MagicDNS suffix is `.trynet`. Change in `settings.json`.
- DERP protocol is a single opcode — no fancy framing, because every payload
  is already a WireGuard packet. See `internal/derp/server.go`.
