// Package control implements the coordination plane: node registration,
// netmap distribution, ACL evaluation, pre-auth key management, and MagicDNS.
package control

import (
	"encoding/json"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/apolzek/trynet/internal/protocol"
)

// Store holds the tailnet state. Persistence is a single JSON file. All
// mutations go through Store so we can bump version + fan out to subscribers.
type Store struct {
	path string

	mu       sync.RWMutex
	data     state
	version  atomic.Uint64

	subscribers map[string]chan struct{} // node key (string) -> wake-up
	subMu       sync.Mutex
}

type state struct {
	Settings  Settings                       `json:"settings"`
	Nodes     map[string]*protocol.Node      `json:"nodes"`      // by node ID
	Keys      map[string]*protocol.PreAuthKey `json:"preauth_keys"` // by secret
	ACL       *protocol.ACLPolicy            `json:"acl"`
	Allocated map[string]bool                `json:"allocated"`  // tailnet IPs in use
}

// Settings holds tailnet-wide knobs.
type Settings struct {
	TailnetName       string        `json:"tailnet_name"`
	IPv4CIDR          string        `json:"ipv4_cidr"`
	DNSSuffix         string        `json:"dns_suffix"`
	RequireApproval   bool          `json:"require_approval"`
	DefaultKeyExpiry  time.Duration `json:"default_key_expiry"`
	DerpURL           string        `json:"derp_url"`
	DerpKey           protocol.Key  `json:"derp_key"`
	AdminToken        string        `json:"admin_token"` // shared secret for UI
}

// NewStore loads an existing state file or starts empty.
func NewStore(path string) (*Store, error) {
	s := &Store{
		path:        path,
		subscribers: map[string]chan struct{}{},
	}
	s.data = state{
		Settings: Settings{
			TailnetName:      "trynet",
			IPv4CIDR:         "100.64.0.0/10",
			DNSSuffix:        "trynet",
			RequireApproval:  false,
			DefaultKeyExpiry: 180 * 24 * time.Hour,
		},
		Nodes:     map[string]*protocol.Node{},
		Keys:      map[string]*protocol.PreAuthKey{},
		ACL:       defaultPolicy(),
		Allocated: map[string]bool{},
	}
	if _, err := os.Stat(path); err == nil {
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(b, &s.data); err != nil {
			return nil, fmt.Errorf("parse state: %w", err)
		}
	}
	s.version.Store(1)
	return s, nil
}

func defaultPolicy() *protocol.ACLPolicy {
	return &protocol.ACLPolicy{
		ACLs: []protocol.ACLRule{
			{Action: "accept", Src: []string{"*"}, Dst: []string{"*:*"}},
		},
		DNS: protocol.DNSConfig{MagicDNS: true, Suffix: "trynet"},
	}
}

func (s *Store) save() error {
	b, err := json.MarshalIndent(&s.data, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// Version returns the current monotonically-increasing netmap version.
func (s *Store) Version() uint64 { return s.version.Load() }

func (s *Store) bump() uint64 { return s.version.Add(1) }

// Settings returns a copy of current tailnet settings.
func (s *Store) SettingsCopy() Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data.Settings
}

// UpdateSettings mutates settings atomically.
func (s *Store) UpdateSettings(mut func(*Settings)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	mut(&s.data.Settings)
	s.bump()
	s.wakeAll()
	return s.save()
}

// -----------------------------------------------------------------------------
// Nodes
// -----------------------------------------------------------------------------

// ListNodes returns a snapshot of all nodes.
func (s *Store) ListNodes() []*protocol.Node {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*protocol.Node, 0, len(s.data.Nodes))
	for _, n := range s.data.Nodes {
		cp := *n
		out = append(out, &cp)
	}
	return out
}

// NodeByKey finds a node by its WireGuard public key.
func (s *Store) NodeByKey(k protocol.Key) *protocol.Node {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, n := range s.data.Nodes {
		if n.NodeKey == k {
			cp := *n
			return &cp
		}
	}
	return nil
}

// NodeByID fetches a node by its ID.
func (s *Store) NodeByID(id string) *protocol.Node {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if n, ok := s.data.Nodes[id]; ok {
		cp := *n
		return &cp
	}
	return nil
}

// UpsertNode inserts or overwrites a node and bumps version.
func (s *Store) UpsertNode(n *protocol.Node) error {
	s.mu.Lock()
	s.data.Nodes[n.ID] = n
	s.bump()
	err := s.save()
	s.mu.Unlock()
	s.wakeAll()
	return err
}

// DeleteNode removes a node and releases its IP.
func (s *Store) DeleteNode(id string) error {
	s.mu.Lock()
	n, ok := s.data.Nodes[id]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("node not found")
	}
	delete(s.data.Nodes, id)
	if n.TailnetIP.IsValid() {
		delete(s.data.Allocated, n.TailnetIP.String())
	}
	s.bump()
	err := s.save()
	s.mu.Unlock()
	s.wakeAll()
	return err
}

// UpdateNodeEndpoints records a node's self-reported endpoints.
func (s *Store) UpdateNodeEndpoints(k protocol.Key, endpoints []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, n := range s.data.Nodes {
		if n.NodeKey == k {
			n.Endpoints = endpoints
			n.LastSeen = time.Now()
			s.bump()
			_ = s.save()
			// fan-out is cheap, just do it outside of the lock.
			go s.wakeAll()
			return
		}
	}
}

// TouchNode updates only LastSeen (called on every poll request).
func (s *Store) TouchNode(k protocol.Key) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, n := range s.data.Nodes {
		if n.NodeKey == k {
			n.LastSeen = time.Now()
			return
		}
	}
}

// AllocateIP reserves the next free tailnet IP from settings.IPv4CIDR. Naive
// linear scan, fine up to a few thousand nodes.
func (s *Store) AllocateIP() (netip.Addr, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	pfx, err := netip.ParsePrefix(s.data.Settings.IPv4CIDR)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("parse cidr: %w", err)
	}
	// skip network + broadcast-ish; start at +2.
	addr := pfx.Addr().Next().Next()
	for pfx.Contains(addr) {
		if !s.data.Allocated[addr.String()] {
			s.data.Allocated[addr.String()] = true
			_ = s.save()
			return addr, nil
		}
		addr = addr.Next()
	}
	return netip.Addr{}, fmt.Errorf("ip pool exhausted")
}

// -----------------------------------------------------------------------------
// Pre-auth keys
// -----------------------------------------------------------------------------

// CreateKey stores a pre-auth key.
func (s *Store) CreateKey(k *protocol.PreAuthKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Keys[k.Secret] = k
	return s.save()
}

// LookupKey validates a key by secret. Returns (*PreAuthKey, ok).
func (s *Store) LookupKey(secret string) (*protocol.PreAuthKey, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	k, ok := s.data.Keys[secret]
	if !ok {
		return nil, false
	}
	if !k.Expires.IsZero() && time.Now().After(k.Expires) {
		return nil, false
	}
	if k.Used && !k.Reusable {
		return nil, false
	}
	cp := *k
	return &cp, true
}

// ListKeys returns all pre-auth keys (admin only).
func (s *Store) ListKeys() []*protocol.PreAuthKey {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*protocol.PreAuthKey, 0, len(s.data.Keys))
	for _, k := range s.data.Keys {
		cp := *k
		out = append(out, &cp)
	}
	return out
}

// MarkKeyUsed flips the `used` flag when a key enrolls a node.
func (s *Store) MarkKeyUsed(secret, nodeID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if k, ok := s.data.Keys[secret]; ok {
		k.Used = true
		k.UsedBy = nodeID
		_ = s.save()
	}
}

// DeleteKey revokes a pre-auth key.
func (s *Store) DeleteKey(secret string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data.Keys, secret)
	_ = s.save()
}

// -----------------------------------------------------------------------------
// ACL
// -----------------------------------------------------------------------------

// ACL returns a copy of the current policy.
func (s *Store) ACL() protocol.ACLPolicy {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.data.ACL == nil {
		return *defaultPolicy()
	}
	return *s.data.ACL
}

// SetACL replaces the policy.
func (s *Store) SetACL(p *protocol.ACLPolicy) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.ACL = p
	s.bump()
	err := s.save()
	go s.wakeAll()
	return err
}

// -----------------------------------------------------------------------------
// Subscribers: the Hub registers a channel per connected node. Any state
// change calls wakeAll, which closes the channel so the long-poll returns.
// -----------------------------------------------------------------------------

// Subscribe registers a wake-up channel for the given node key. Returns an
// unsubscribe function.
func (s *Store) Subscribe(nodeKey protocol.Key) (<-chan struct{}, func()) {
	ch := make(chan struct{}, 1)
	id := nodeKey.String()
	s.subMu.Lock()
	s.subscribers[id] = ch
	s.subMu.Unlock()
	return ch, func() {
		s.subMu.Lock()
		if cur, ok := s.subscribers[id]; ok && cur == ch {
			delete(s.subscribers, id)
		}
		s.subMu.Unlock()
	}
}

func (s *Store) wakeAll() {
	s.subMu.Lock()
	defer s.subMu.Unlock()
	for _, ch := range s.subscribers {
		select {
		case ch <- struct{}{}:
		default:
			// already pending wake; that's fine
		}
	}
}
