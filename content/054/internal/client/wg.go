package client

import (
	"fmt"
	"net"
	"net/netip"
	"os/exec"
	"time"

	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/apolzek/trynet/internal/protocol"
)

// WGController wraps a wgctrl.Client and knows how to reconcile the kernel
// WireGuard state from a trynet NetMap.
type WGController struct {
	ifName string
	c      *wgctrl.Client
}

// NewWGController opens the wgctrl netlink socket.
func NewWGController(ifName string) (*WGController, error) {
	c, err := wgctrl.New()
	if err != nil {
		return nil, fmt.Errorf("wgctrl: %w", err)
	}
	return &WGController{ifName: ifName, c: c}, nil
}

// Close releases the netlink socket.
func (w *WGController) Close() error { return w.c.Close() }

// EnsureInterface creates the wg0 device if missing and assigns the given
// tailnet IP to it. Uses `ip` because wgctrl won't create links.
func (w *WGController) EnsureInterface(addr netip.Addr) error {
	if _, err := net.InterfaceByName(w.ifName); err != nil {
		// `ip link add dev wg0 type wireguard`
		if out, err := exec.Command("ip", "link", "add", "dev", w.ifName, "type", "wireguard").CombinedOutput(); err != nil {
			return fmt.Errorf("create link: %v: %s", err, out)
		}
	}
	// flush any old address, set ours, bring up
	_ = exec.Command("ip", "addr", "flush", "dev", w.ifName).Run()
	cidr := fmt.Sprintf("%s/32", addr.String())
	if out, err := exec.Command("ip", "addr", "add", cidr, "dev", w.ifName).CombinedOutput(); err != nil {
		return fmt.Errorf("addr add %s: %v: %s", cidr, err, out)
	}
	if out, err := exec.Command("ip", "link", "set", "dev", w.ifName, "up").CombinedOutput(); err != nil {
		return fmt.Errorf("link up: %v: %s", err, out)
	}
	return nil
}

// Configure sets the private key, listen port, and peers on the interface.
func (w *WGController) Configure(privKey protocol.Key, listenPort int, peers []protocol.PeerView) error {
	var pk wgtypes.Key
	copy(pk[:], privKey[:])

	peerCfgs := make([]wgtypes.PeerConfig, 0, len(peers))
	for _, p := range peers {
		var pubK wgtypes.Key
		copy(pubK[:], p.NodeKey[:])

		var endpoint *net.UDPAddr
		if len(p.Endpoints) > 0 {
			if addr, err := net.ResolveUDPAddr("udp", p.Endpoints[0]); err == nil {
				endpoint = addr
			}
		}
		allowed := toIPNets(p.AllowedIPs)
		ka := 25 * time.Second
		peerCfgs = append(peerCfgs, wgtypes.PeerConfig{
			PublicKey:                   pubK,
			ReplaceAllowedIPs:           true,
			AllowedIPs:                  allowed,
			Endpoint:                    endpoint,
			PersistentKeepaliveInterval: &ka,
		})
	}

	lp := listenPort
	cfg := wgtypes.Config{
		PrivateKey:   &pk,
		ListenPort:   &lp,
		ReplacePeers: true,
		Peers:        peerCfgs,
	}
	return w.c.ConfigureDevice(w.ifName, cfg)
}

// InstallRoutes adds or refreshes kernel routes pointing at wg0 for each
// AllowedIP range that isn't /32 (those are handled by the interface address).
func (w *WGController) InstallRoutes(peers []protocol.PeerView) error {
	for _, p := range peers {
		for _, pfx := range p.AllowedIPs {
			if pfx.Bits() == pfx.Addr().BitLen() {
				continue // /32 or /128: on-link via wg0
			}
			// `ip route replace <pfx> dev wg0`
			if out, err := exec.Command("ip", "route", "replace", pfx.String(), "dev", w.ifName).CombinedOutput(); err != nil {
				return fmt.Errorf("route %s: %v: %s", pfx, err, out)
			}
		}
	}
	return nil
}

// LocalEndpoints tries to enumerate UDP endpoints this node is reachable on.
// It does a best-effort scan of non-loopback addresses + the wg listen port.
func LocalEndpoints(listenPort int) []string {
	ifs, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var out []string
	for _, ifc := range ifs {
		if ifc.Flags&net.FlagUp == 0 || ifc.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := ifc.Addrs()
		for _, a := range addrs {
			ip, _, err := net.ParseCIDR(a.String())
			if err != nil {
				continue
			}
			if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
				continue
			}
			out = append(out, fmt.Sprintf("%s:%d", ip.String(), listenPort))
		}
	}
	return out
}

// TearDown removes wg0 entirely. Called on logout.
func (w *WGController) TearDown() error {
	if _, err := net.InterfaceByName(w.ifName); err != nil {
		return nil
	}
	out, err := exec.Command("ip", "link", "del", "dev", w.ifName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("del link: %v: %s", err, out)
	}
	return nil
}

func toIPNets(ps []netip.Prefix) []net.IPNet {
	out := make([]net.IPNet, 0, len(ps))
	for _, p := range ps {
		_, ipnet, err := net.ParseCIDR(p.String())
		if err == nil {
			out = append(out, *ipnet)
		}
	}
	return out
}
