// Package protocol defines the wire types shared between control, derp,
// daemon, and ui. Everything that crosses a process boundary is JSON.
package protocol

import (
	"encoding/json"
	"net/netip"
	"time"
)

// Key is a Curve25519 public (or private) key, base64-encoded without padding
// when serialized. 32 raw bytes.
type Key [32]byte

func (k Key) String() string            { return encodeKey(k) }
func (k Key) MarshalJSON() ([]byte, error) { return json.Marshal(k.String()) }
func (k *Key) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	out, err := decodeKey(s)
	if err != nil {
		return err
	}
	*k = out
	return nil
}

// Node is the control-plane record for a single machine in the tailnet.
type Node struct {
	ID        string       `json:"id"`
	Name      string       `json:"name"`            // user-visible hostname
	User      string       `json:"user"`            // owning identity
	Tags      []string     `json:"tags,omitempty"`  // e.g. "tag:server"
	NodeKey   Key          `json:"node_key"`        // wg public key
	TailnetIP netip.Addr   `json:"tailnet_ip"`
	Endpoints []string     `json:"endpoints,omitempty"` // "ip:port" candidates
	Routes    []netip.Prefix `json:"routes,omitempty"` // advertised subnets
	ExitNode  bool         `json:"exit_node,omitempty"`
	OS        string       `json:"os,omitempty"`
	Created   time.Time    `json:"created"`
	LastSeen  time.Time    `json:"last_seen"`
	Expiry    time.Time    `json:"expiry"`          // zero = never
	Approved  bool         `json:"approved"`
}

// Online is a best-effort freshness check.
func (n *Node) Online() bool { return time.Since(n.LastSeen) < 2*time.Minute }

// Expired reports whether the node's key has crossed its expiry.
func (n *Node) Expired() bool {
	return !n.Expiry.IsZero() && time.Now().After(n.Expiry)
}

// PreAuthKey is a token an operator generates so a headless node can enroll
// without going through an interactive login.
type PreAuthKey struct {
	ID        string    `json:"id"`
	Secret    string    `json:"secret"`              // opaque bearer token
	User      string    `json:"user"`
	Tags      []string  `json:"tags,omitempty"`
	Reusable  bool      `json:"reusable"`
	Ephemeral bool      `json:"ephemeral,omitempty"` // node auto-removed on disconnect
	Created   time.Time `json:"created"`
	Expires   time.Time `json:"expires"`
	Used      bool      `json:"used,omitempty"`
	UsedBy    string    `json:"used_by,omitempty"`
}

// ACLPolicy is the tailnet-wide access policy. Rules are evaluated in order;
// the first `accept` rule that matches the (src, dst, port) triple wins.
// Anything not matched is denied.
type ACLPolicy struct {
	Groups map[string][]string `json:"groups,omitempty"` // "group:eng" -> [users]
	Tags   map[string]string   `json:"tagOwners,omitempty"` // "tag:prod" -> "group:eng"
	ACLs   []ACLRule           `json:"acls"`
	SSH    []SSHRule           `json:"ssh,omitempty"`
	DNS    DNSConfig           `json:"dns,omitempty"`
}

type ACLRule struct {
	Action string   `json:"action"` // "accept" for now
	Src    []string `json:"src"`    // user, tag, group, CIDR, or "*"
	Dst    []string `json:"dst"`    // "tag:foo:22", "100.64.0.0/10:*", "*:443"
	Proto  string   `json:"proto,omitempty"` // "tcp", "udp", "icmp", "" = any
}

type SSHRule struct {
	Action string   `json:"action"` // "accept-ssh"
	Src    []string `json:"src"`
	Dst    []string `json:"dst"`
	Users  []string `json:"users"` // unix users allowed as targets
}

type DNSConfig struct {
	MagicDNS    bool              `json:"magic_dns"`
	Suffix      string            `json:"suffix"`         // "trynet"
	Nameservers []string          `json:"nameservers,omitempty"`
	Routes      map[string][]string `json:"routes,omitempty"` // split DNS
}

// -----------------------------------------------------------------------------
// API messages
// -----------------------------------------------------------------------------

// RegisterRequest is POSTed by a freshly-started node.
type RegisterRequest struct {
	NodeKey    Key      `json:"node_key"`
	Hostname   string   `json:"hostname"`
	OS         string   `json:"os"`
	AuthKey    string   `json:"auth_key"`           // pre-auth token
	Endpoints  []string `json:"endpoints,omitempty"`
	Routes     []string `json:"routes,omitempty"`   // CIDR strings
	ExitNode   bool     `json:"exit_node,omitempty"`
}

type RegisterResponse struct {
	NodeID    string     `json:"node_id"`
	TailnetIP netip.Addr `json:"tailnet_ip"`
	NetMask   int        `json:"netmask"`   // /32 for tailnet IPs
	Expiry    time.Time  `json:"expiry"`
}

// PollRequest is used for long-poll netmap retrieval.
type PollRequest struct {
	NodeKey        Key      `json:"node_key"`
	KnownVersion   uint64   `json:"known_version"`
	Endpoints      []string `json:"endpoints,omitempty"`
}

// NetMap is the view one node has of the tailnet.
type NetMap struct {
	Version  uint64     `json:"version"`
	Self     PeerView   `json:"self"`
	Peers    []PeerView `json:"peers"`
	DNS      DNSConfig  `json:"dns"`
	Hosts    map[string]netip.Addr `json:"hosts"`      // MagicDNS table
	DerpURL  string     `json:"derp_url,omitempty"`
	DerpKey  Key        `json:"derp_key,omitempty"`     // relay server's identity
	UserName string     `json:"user_name"`
}

// PeerView is a node as seen by another node: only the fields a peer needs to
// route traffic, and only peers ACLs allowed it to see.
type PeerView struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	NodeKey   Key            `json:"node_key"`
	TailnetIP netip.Addr     `json:"tailnet_ip"`
	Endpoints []string       `json:"endpoints,omitempty"`
	Routes    []netip.Prefix `json:"routes,omitempty"`
	ExitNode  bool           `json:"exit_node,omitempty"`
	Online    bool           `json:"online"`
	AllowedIPs []netip.Prefix `json:"allowed_ips"`
}

// EndpointsReport is how a node reports its currently-observed UDP endpoints
// (its own best guess, no server-side STUN yet).
type EndpointsReport struct {
	NodeKey   Key      `json:"node_key"`
	Endpoints []string `json:"endpoints"`
}

// -----------------------------------------------------------------------------
// Local CLI ↔ daemon protocol (over UNIX socket)
// -----------------------------------------------------------------------------

type LocalCommand struct {
	Op      string          `json:"op"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type LocalResponse struct {
	OK    bool            `json:"ok"`
	Error string          `json:"error,omitempty"`
	Data  json.RawMessage `json:"data,omitempty"`
}

type LocalStatus struct {
	Running    bool       `json:"running"`
	BackendState string   `json:"backend_state"` // "Stopped","NoState","NeedsLogin","Starting","Running"
	TailnetIP  netip.Addr `json:"tailnet_ip"`
	NodeKey    Key        `json:"node_key"`
	Peers      []PeerView `json:"peers"`
	Hosts      map[string]netip.Addr `json:"hosts"`
	LastError  string     `json:"last_error,omitempty"`
}
