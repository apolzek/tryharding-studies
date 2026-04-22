package control

import (
	"fmt"
	"net/netip"
	"strings"

	"github.com/apolzek/trynet/internal/protocol"
)

// BuildNetMap computes the per-node NetMap that this node should receive.
// ACLs filter the peer list so a node only learns about peers it's allowed to
// talk to — both to shrink the map and to avoid leaking metadata.
func (s *Store) BuildNetMap(nodeKey protocol.Key) (*protocol.NetMap, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var self *protocol.Node
	for _, n := range s.data.Nodes {
		if n.NodeKey == nodeKey {
			cp := *n
			self = &cp
			break
		}
	}
	if self == nil {
		return nil, fmt.Errorf("unknown node")
	}
	if !self.Approved && s.data.Settings.RequireApproval {
		// return an empty netmap — node is known but gated
		return &protocol.NetMap{
			Version:  s.version.Load(),
			Self:     toPeerView(self, nil),
			Peers:    nil,
			DNS:      s.data.ACL.DNS,
			Hosts:    map[string]netip.Addr{},
			DerpURL:  s.data.Settings.DerpURL,
			DerpKey:  s.data.Settings.DerpKey,
			UserName: self.User,
		}, nil
	}

	engine := newACLEngine(*s.data.ACL)

	var all []*protocol.Node
	for _, n := range s.data.Nodes {
		if n.Expired() {
			continue
		}
		all = append(all, n)
	}
	allowed := engine.allowedPeers(self, all)

	peers := make([]protocol.PeerView, 0, len(allowed))
	hosts := map[string]netip.Addr{}

	for _, p := range allowed {
		// AllowedIPs = peer's tailnet IP /32 + any advertised routes that
		// aren't served by a closer peer. Simplistic: we hand every route out.
		allowedIPs := []netip.Prefix{
			netip.PrefixFrom(p.TailnetIP, 32),
		}
		allowedIPs = append(allowedIPs, p.Routes...)
		if p.ExitNode {
			// exit-node routes — client decides whether to actually use them
			allowedIPs = append(allowedIPs,
				netip.MustParsePrefix("0.0.0.0/0"),
				netip.MustParsePrefix("::/0"),
			)
		}
		peers = append(peers, toPeerView(p, allowedIPs))
		hosts[dnsName(p.Name, s.data.Settings.DNSSuffix)] = p.TailnetIP
	}

	// include self in hosts so `ping self-name` works.
	hosts[dnsName(self.Name, s.data.Settings.DNSSuffix)] = self.TailnetIP

	return &protocol.NetMap{
		Version:  s.version.Load(),
		Self:     toPeerView(self, []netip.Prefix{netip.PrefixFrom(self.TailnetIP, 32)}),
		Peers:    peers,
		DNS:      s.data.ACL.DNS,
		Hosts:    hosts,
		DerpURL:  s.data.Settings.DerpURL,
		DerpKey:  s.data.Settings.DerpKey,
		UserName: self.User,
	}, nil
}

func toPeerView(n *protocol.Node, allowed []netip.Prefix) protocol.PeerView {
	return protocol.PeerView{
		ID:         n.ID,
		Name:       n.Name,
		NodeKey:    n.NodeKey,
		TailnetIP:  n.TailnetIP,
		Endpoints:  n.Endpoints,
		Routes:     n.Routes,
		ExitNode:   n.ExitNode,
		Online:     n.Online(),
		AllowedIPs: allowed,
	}
}

// dnsName sanitises a hostname and appends the tailnet suffix.
func dnsName(name, suffix string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")
	if suffix == "" {
		return name
	}
	return name + "." + suffix
}
