# Tailscale feature analysis & trynet scope

`trynet` is an open-source POC of a Tailscale-like mesh VPN. This document
surveys Tailscale's feature surface and states what trynet implements, what it
stubs, and what it intentionally leaves out.

## 1. Tailscale feature surface

| Category | Feature | What it does |
|---|---|---|
| Core | WireGuard mesh | Each node peers directly with every other node over WireGuard. |
| Core | Coordination server | Central service (`login.tailscale.com`) that handles identity, key exchange, netmap distribution. |
| Core | DERP relays | Geographically distributed TLS relays that shuttle encrypted packets when direct P2P is impossible. |
| Core | NAT traversal | STUN-based endpoint discovery + UDP hole punching + DERP fallback. |
| Core | Key rotation | Periodic WireGuard session-key rotation without re-auth. |
| Identity | SSO login | OAuth/OIDC (Google, GitHub, Okta, etc.) — users are authenticated by an IdP, not local accounts. |
| Identity | Pre-auth keys | Long-lived tokens used to enroll headless/automated nodes. |
| Identity | Device approval | Admin must manually approve new devices (optional tailnet-wide setting). |
| Identity | Tailnet lock | Cryptographic signing that prevents the coordination server from adding malicious nodes to the tailnet. |
| Identity | Key expiry | Node keys expire after N days unless refreshed. |
| Networking | MagicDNS | Short hostnames (`laptop`, `server`) resolve to tailnet IPs via a tailnet-local DNS. |
| Networking | Split DNS | Override resolution for specific domains per tailnet. |
| Networking | Subnet routes | A node advertises a CIDR it can reach, turning it into a gateway for that subnet. |
| Networking | Exit nodes | A node can volunteer as the default-route egress for the tailnet. |
| Networking | IPv6 (ULA) | Each tailnet gets a ULA /48, each node a /128. |
| Networking | 4via6 | Embed IPv4 subnets inside IPv6 to solve overlapping RFC1918 ranges. |
| Policy | ACLs (HuJSON) | Rule-based "who can talk to whom, on which ports". Evaluated by coordination server before netmap is pushed. |
| Policy | Groups / tags | Named sets of users or nodes; ACLs reference them. |
| Policy | SSH ACLs | ACLs can gate Tailscale SSH. |
| Features | Taildrop | Send files between nodes you own. |
| Features | Tailscale SSH | SSH server/client that authenticates via tailnet identity instead of keys/passwords. |
| Features | Tailscale Serve | Reverse-proxy HTTPS service on a node's tailnet IP. |
| Features | Tailscale Funnel | Expose a local service publicly (`*.ts.net`) with TLS. |
| Features | User switching | One daemon, multiple tailnets. |
| Features | Mobile clients | iOS/Android apps using WireGuard-go. |
| Features | Auto-updates | Managed client self-update. |
| Ops | Admin UI | Web console to list nodes, edit ACLs, manage keys, review logs. |
| Ops | Webhooks / audit log | Stream events externally. |
| Ops | SCIM | Provision users from IdP. |

## 2. trynet implementation status

Legend: `✅` working code · `🟡` scaffolded (endpoints/types exist, partial logic) · `🛑` explicitly out of scope for this POC.

| Feature | Status | Notes |
|---|---|---|
| WireGuard mesh | ✅ | Client drives `wgctrl` to configure peers from the netmap. |
| Coordination server | ✅ | HTTP API: register, poll netmap (long-poll), report endpoints. JSON on disk. |
| DERP relay | ✅ | Custom WebSocket relay (not wire-compatible with Tailscale's DERP). Clients open a persistent session; server routes by destination node key. |
| NAT traversal | 🟡 | Endpoint discovery via self-report (STUN-lite over the control channel). Direct UDP first, DERP fallback. No hole punching. |
| Key rotation | 🟡 | Control rotates WireGuard keys on each client re-registration. No in-session rekey. |
| SSO login | 🛑 | Replaced by pre-auth keys + admin tokens. Pluggable `authenticator` interface leaves room for OIDC. |
| Pre-auth keys | ✅ | Admin UI generates reusable or one-shot keys with expiry and tag assignment. |
| Device approval | ✅ | Tailnet setting; unapproved nodes get an empty netmap. |
| Tailnet lock | 🛑 | Documented as future work. |
| Key expiry | ✅ | Node keys carry an `Expiry` field; expired nodes are excluded from peers' netmaps. |
| MagicDNS | ✅ | Client writes managed block into `/etc/hosts`. Control owns the name→IP mapping. |
| Split DNS | 🟡 | Config field present, resolver not implemented (would require DNS proxy). |
| Subnet routes | ✅ | Node advertises CIDRs; control validates against ACL; peers install routes. |
| Exit nodes | ✅ | Same mechanism as subnet routes with `0.0.0.0/0`. Client has `--exit-node` flag. |
| IPv6 ULA | 🟡 | Addresses allocated from `fd7a:trynet::/48`, but most features exercised over IPv4. |
| 4via6 | 🛑 | Out of scope. |
| ACLs | ✅ | JSON policy: `src` (user/tag/group) → `dst` (tag/host:port). Evaluated when building netmaps. |
| Groups / tags | ✅ | First-class; nodes carry tags, ACLs and pre-auth keys reference them. |
| SSH ACLs | 🟡 | ACL schema has `action: "accept-ssh"`; enforcement is TODO in the SSH module. |
| Taildrop | ✅ | Client exposes HTTP endpoint on tailnet IP for peer-authenticated file push. |
| trynet SSH | 🛑 | Stub. |
| Serve / Funnel | 🛑 | Stub. |
| User switching | 🛑 | One identity per daemon. |
| Mobile clients | 🛑 | Desktop Linux only. |
| Admin UI | ✅ | Server-rendered Go templates: nodes, keys, ACLs, settings. |
| Audit log | 🟡 | In-memory ring buffer, surfaced in UI. |
| SCIM | 🛑 | Out of scope. |

## 3. Design differences vs Tailscale

- **Protocol is not wire-compatible.** trynet defines its own JSON API and
  relay protocol — a Tailscale client cannot talk to trynet and vice-versa.
  This keeps the codebase small enough to read in a weekend.
- **No OIDC.** Tailscale's product is really "WireGuard + IdP". For a
  self-hosted POC we rely on admin-provisioned pre-auth keys and leave OIDC
  behind an `authenticator` interface.
- **JSON-on-disk storage.** No SQLite/Postgres. The control server loads the
  tailnet state into memory on boot and fsyncs after mutations. Good for
  hundreds of nodes, not for ten thousand.
- **Go stdlib only + `wgctrl`.** No Gin, no GORM, no cobra. Easier to audit,
  fewer moving parts.

## 4. What you still have to bring

- A VPS with a public IP and open UDP/41641 (WireGuard), TCP/443 (control +
  UI), TCP/3478 (relay). Docker Compose handles the rest.
- Root on each client (WireGuard needs `CAP_NET_ADMIN`).
- Kernel with WireGuard (≥ 5.6) or `wireguard-go` userspace fallback.
