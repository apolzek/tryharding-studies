package control

import (
	"net/netip"
	"strconv"
	"strings"

	"github.com/apolzek/trynet/internal/protocol"
)

// aclEngine evaluates a policy against a (src node, dst node, dst port) triple.
// Implementation is deliberately simple: linear scan through rules, first
// match wins. Destinations are expressed as "<target>:<port>", where target is:
//   - "*"
//   - "tag:foo"
//   - a CIDR
//   - a MagicDNS hostname (resolved via the hosts map)
//
// Sources are:
//   - "*"
//   - "<user>@"
//   - "tag:foo"
//   - "group:eng"
//   - a CIDR
type aclEngine struct {
	policy protocol.ACLPolicy
}

func newACLEngine(p protocol.ACLPolicy) *aclEngine { return &aclEngine{policy: p} }

// allowedPeers returns, for a given node, the subset of peers it is allowed to
// originate traffic to, with the matching AllowedIPs that should end up in its
// WireGuard config.
func (e *aclEngine) allowedPeers(self *protocol.Node, peers []*protocol.Node) []*protocol.Node {
	var out []*protocol.Node
	for _, p := range peers {
		if p.ID == self.ID {
			continue
		}
		if e.canConnect(self, p) {
			out = append(out, p)
		}
	}
	return out
}

// canConnect is true if the policy has any `accept` rule where self matches
// src and p matches dst on any port.
func (e *aclEngine) canConnect(self, p *protocol.Node) bool {
	for _, rule := range e.policy.ACLs {
		if rule.Action != "accept" {
			continue
		}
		if !e.matchesSrc(rule.Src, self) {
			continue
		}
		for _, dst := range rule.Dst {
			target, _ := splitHostPort(dst)
			if e.matchesTarget(target, p) {
				return true
			}
		}
	}
	return false
}

func (e *aclEngine) matchesSrc(srcs []string, n *protocol.Node) bool {
	for _, s := range srcs {
		if s == "*" {
			return true
		}
		if strings.HasPrefix(s, "tag:") {
			for _, t := range n.Tags {
				if t == s {
					return true
				}
			}
			continue
		}
		if strings.HasPrefix(s, "group:") {
			for _, u := range e.policy.Groups[s] {
				if u == n.User {
					return true
				}
			}
			continue
		}
		if strings.HasSuffix(s, "@") {
			if n.User == strings.TrimSuffix(s, "@") {
				return true
			}
			continue
		}
		if pfx, err := netip.ParsePrefix(s); err == nil {
			if pfx.Contains(n.TailnetIP) {
				return true
			}
		}
	}
	return false
}

func (e *aclEngine) matchesTarget(target string, n *protocol.Node) bool {
	if target == "*" {
		return true
	}
	if strings.HasPrefix(target, "tag:") {
		for _, t := range n.Tags {
			if t == target {
				return true
			}
		}
		return false
	}
	if pfx, err := netip.ParsePrefix(target); err == nil {
		if pfx.Contains(n.TailnetIP) {
			return true
		}
		for _, r := range n.Routes {
			if pfx.Overlaps(r) {
				return true
			}
		}
		return false
	}
	// treat as hostname
	return target == n.Name
}

func splitHostPort(s string) (host, port string) {
	i := strings.LastIndex(s, ":")
	if i < 0 {
		return s, ""
	}
	return s[:i], s[i+1:]
}

// portAllowed is there for future SSH/port-level filtering. Not consulted when
// building the netmap — mesh routing is currently all-or-nothing per peer.
func portAllowed(p string, want int) bool {
	if p == "*" || p == "" {
		return true
	}
	for _, chunk := range strings.Split(p, ",") {
		if chunk == strconv.Itoa(want) {
			return true
		}
		if i := strings.Index(chunk, "-"); i > 0 {
			lo, _ := strconv.Atoi(chunk[:i])
			hi, _ := strconv.Atoi(chunk[i+1:])
			if want >= lo && want <= hi {
				return true
			}
		}
	}
	return false
}
