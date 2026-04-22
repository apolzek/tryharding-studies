// Package client is the on-device agent ("trynetd"): it keeps WireGuard in
// sync with the control plane's netmap, manages MagicDNS, and exposes a UNIX
// socket so the CLI can drive it.
package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/netip"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/apolzek/trynet/internal/crypto"
	"github.com/apolzek/trynet/internal/protocol"
)

// Agent is the long-running daemon state.
type Agent struct {
	log       *log.Logger
	cfg       *Config
	statePath string
	ifName    string

	mu    sync.Mutex
	state *State
	ctl   *controlClient
	wg    *WGController

	// Current netmap (for status).
	netmap    *protocol.NetMap
	backend   string // "Stopped","NoState","NeedsLogin","Starting","Running"
	lastError string

	cancelLoop context.CancelFunc
}

// NewAgent constructs an agent from config paths.
func NewAgent(cfg *Config, statePath, ifName string, lg *log.Logger) (*Agent, error) {
	if lg == nil {
		lg = log.Default()
	}
	a := &Agent{
		log:       lg,
		cfg:       cfg,
		statePath: statePath,
		ifName:    ifName,
		ctl:       newControlClient(cfg.ControlURL, cfg.Insecure),
		backend:   "NoState",
	}
	st, err := LoadState(statePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	} else {
		a.state = st
		a.backend = "NeedsLogin"
		if st.AuthKey != "" && !st.NodeKeyZero() {
			// we have enough to try re-registering silently
			a.backend = "Starting"
		}
	}
	return a, nil
}

// NodeKeyZero reports whether a persisted state has a non-zero public key.
func (s *State) NodeKeyZero() bool {
	var zero protocol.Key
	return s.PublicKey == zero
}

// Start brings the agent online: if state is present, re-register; otherwise
// wait for a CLI `up`.
func (a *Agent) Start(ctx context.Context) error {
	if a.state != nil && a.state.AuthKey != "" && !a.state.NodeKeyZero() {
		go a.runMain(ctx, a.state.AuthKey, "", false)
	}
	<-ctx.Done()
	return nil
}

// LoginOptions is what the CLI's `up` carries.
type LoginOptions struct {
	AuthKey   string
	Hostname  string
	Routes    []string
	ExitNode  bool
}

// Login performs initial registration and kicks off the main loop.
func (a *Agent) Login(ctx context.Context, opts LoginOptions) error {
	a.mu.Lock()
	if a.state == nil {
		priv, pub, err := crypto.GenerateKeyPair()
		if err != nil {
			a.mu.Unlock()
			return fmt.Errorf("keygen: %w", err)
		}
		a.state = &State{
			PrivateKey: priv,
			PublicKey:  pub,
			ControlURL: a.cfg.ControlURL,
			RelayURL:   a.cfg.RelayURL,
			Insecure:   a.cfg.Insecure,
			ListenPort: a.cfg.ListenPort,
		}
	}
	a.state.AuthKey = opts.AuthKey
	if err := SaveState(a.statePath, a.state); err != nil {
		a.mu.Unlock()
		return err
	}
	a.mu.Unlock()

	// Restart the main loop so it uses the new auth key.
	if a.cancelLoop != nil {
		a.cancelLoop()
	}
	loopCtx, cancel := context.WithCancel(ctx)
	a.mu.Lock()
	a.cancelLoop = cancel
	a.mu.Unlock()
	go a.runMain(loopCtx, opts.AuthKey, opts.Hostname, opts.ExitNode)
	return nil
}

// Logout tears everything down and forgets the node.
func (a *Agent) Logout(ctx context.Context) error {
	a.mu.Lock()
	if a.cancelLoop != nil {
		a.cancelLoop()
		a.cancelLoop = nil
	}
	st := a.state
	a.mu.Unlock()

	if st != nil {
		_ = a.ctl.Logout(ctx, st.PublicKey)
	}
	if a.wg != nil {
		_ = a.wg.TearDown()
	}
	_ = RemoveHosts()
	_ = os.Remove(a.statePath)

	a.mu.Lock()
	a.state = nil
	a.netmap = nil
	a.backend = "NoState"
	a.mu.Unlock()
	return nil
}

// Down brings the interface offline without forgetting credentials.
func (a *Agent) Down(ctx context.Context) error {
	a.mu.Lock()
	if a.cancelLoop != nil {
		a.cancelLoop()
		a.cancelLoop = nil
	}
	a.backend = "Stopped"
	a.mu.Unlock()

	if a.wg != nil {
		_ = a.wg.TearDown()
	}
	_ = RemoveHosts()
	return nil
}

// Up re-activates an existing session using the stored auth key.
func (a *Agent) Up(ctx context.Context) error {
	a.mu.Lock()
	if a.state == nil || a.state.AuthKey == "" {
		a.mu.Unlock()
		return errors.New("not logged in; use `trynet up --authkey ...`")
	}
	key := a.state.AuthKey
	a.mu.Unlock()
	return a.Login(ctx, LoginOptions{AuthKey: key})
}

// Status returns a snapshot suitable for CLI consumption.
func (a *Agent) Status() protocol.LocalStatus {
	a.mu.Lock()
	defer a.mu.Unlock()
	st := protocol.LocalStatus{
		BackendState: a.backend,
		Running:      a.backend == "Running",
		LastError:    a.lastError,
	}
	if a.state != nil {
		st.NodeKey = a.state.PublicKey
		if ip, err := netip.ParseAddr(a.state.TailnetIP); err == nil {
			st.TailnetIP = ip
		}
	}
	if a.netmap != nil {
		st.Peers = a.netmap.Peers
		st.Hosts = a.netmap.Hosts
	}
	return st
}

// -------------------------------------------------------------------------
// main loop: register, poll netmap forever, report endpoints periodically.
// -------------------------------------------------------------------------

func (a *Agent) runMain(ctx context.Context, authKey, overrideHostname string, exitNode bool) {
	a.setBackend("Starting")

	wg, err := NewWGController(a.ifName)
	if err != nil {
		a.fail(fmt.Errorf("wg open: %w", err))
		return
	}
	a.mu.Lock()
	a.wg = wg
	a.mu.Unlock()

	hn := overrideHostname
	if hn == "" {
		hn, _ = os.Hostname()
	}

	a.mu.Lock()
	st := a.state
	a.mu.Unlock()
	if st == nil {
		a.fail(errors.New("no state"))
		return
	}

	endpoints := LocalEndpoints(st.ListenPort)

	regReq := &protocol.RegisterRequest{
		NodeKey:   st.PublicKey,
		Hostname:  hn,
		OS:        "linux",
		AuthKey:   authKey,
		Endpoints: endpoints,
		ExitNode:  exitNode,
	}
	regResp, err := a.ctl.Register(ctx, regReq)
	if err != nil {
		a.fail(fmt.Errorf("register: %w", err))
		return
	}
	a.log.Printf("registered as %s, IP=%s", regResp.NodeID, regResp.TailnetIP)

	a.mu.Lock()
	a.state.NodeID = regResp.NodeID
	a.state.TailnetIP = regResp.TailnetIP.String()
	_ = SaveState(a.statePath, a.state)
	a.mu.Unlock()

	if err := wg.EnsureInterface(regResp.TailnetIP); err != nil {
		a.fail(fmt.Errorf("ifup: %w", err))
		return
	}

	var known uint64
	for {
		if ctx.Err() != nil {
			return
		}
		pollReq := &protocol.PollRequest{
			NodeKey:      st.PublicKey,
			KnownVersion: known,
			Endpoints:    endpoints,
		}
		nm, err := a.ctl.PollNetMap(ctx, pollReq)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			a.log.Printf("poll: %v (retry 5s)", err)
			time.Sleep(5 * time.Second)
			continue
		}
		if nm.Version != known {
			if err := a.applyNetMap(nm); err != nil {
				a.log.Printf("apply: %v", err)
			} else {
				a.setBackend("Running")
				known = nm.Version
			}
		}
	}
}

func (a *Agent) applyNetMap(nm *protocol.NetMap) error {
	a.mu.Lock()
	a.netmap = nm
	st := a.state
	a.mu.Unlock()

	if err := a.wg.Configure(st.PrivateKey, st.ListenPort, nm.Peers); err != nil {
		return fmt.Errorf("wg configure: %w", err)
	}
	if err := a.wg.InstallRoutes(nm.Peers); err != nil {
		return fmt.Errorf("routes: %w", err)
	}
	if nm.DNS.MagicDNS && len(nm.Hosts) > 0 {
		if err := ApplyHosts(nm.Hosts); err != nil {
			return fmt.Errorf("dns: %w", err)
		}
	}
	return nil
}

func (a *Agent) setBackend(s string) {
	a.mu.Lock()
	a.backend = s
	a.mu.Unlock()
}

func (a *Agent) fail(err error) {
	a.log.Printf("agent failed: %v", err)
	a.mu.Lock()
	a.backend = "NeedsLogin"
	a.lastError = err.Error()
	a.mu.Unlock()
}

// PrettyStatus renders LocalStatus as a compact human string.
func PrettyStatus(st protocol.LocalStatus) string {
	var b strings.Builder
	fmt.Fprintf(&b, "state: %s\n", st.BackendState)
	if st.TailnetIP.IsValid() {
		fmt.Fprintf(&b, "ip:    %s\n", st.TailnetIP)
	}
	if st.LastError != "" {
		fmt.Fprintf(&b, "error: %s\n", st.LastError)
	}
	if len(st.Peers) > 0 {
		fmt.Fprintln(&b, "peers:")
		for _, p := range st.Peers {
			status := "offline"
			if p.Online {
				status = "online"
			}
			fmt.Fprintf(&b, "  %-24s %-16s %s\n", p.Name, p.TailnetIP.String(), status)
		}
	}
	return b.String()
}

// -------------------------------------------------------------------------
// UNIX socket server for CLI
// -------------------------------------------------------------------------

// ServeLocal listens on the UNIX socket for CLI commands. Blocks until ctx is done.
func (a *Agent) ServeLocal(ctx context.Context, sockPath string) error {
	_ = os.Remove(sockPath)
	if err := os.MkdirAll("/run", 0o755); err == nil {
		// best-effort; /run already exists on systemd boxes
	}
	lis, err := listenUnix(sockPath)
	if err != nil {
		return err
	}
	defer lis.Close()
	a.log.Printf("local socket: %s", sockPath)

	go func() {
		<-ctx.Done()
		_ = lis.Close()
	}()

	for {
		c, err := lis.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		go a.handleLocal(ctx, c)
	}
}

func (a *Agent) handleLocal(ctx context.Context, c localConn) {
	defer c.Close()
	dec := json.NewDecoder(c)
	enc := json.NewEncoder(c)
	var cmd protocol.LocalCommand
	if err := dec.Decode(&cmd); err != nil {
		_ = enc.Encode(protocol.LocalResponse{Error: err.Error()})
		return
	}

	resp := protocol.LocalResponse{OK: true}
	switch cmd.Op {
	case "status":
		st := a.Status()
		b, _ := json.Marshal(st)
		resp.Data = b
	case "up":
		var opts LoginOptions
		if len(cmd.Payload) > 0 {
			_ = json.Unmarshal(cmd.Payload, &opts)
		}
		if opts.AuthKey == "" {
			if err := a.Up(ctx); err != nil {
				resp.OK = false
				resp.Error = err.Error()
			}
		} else {
			if err := a.Login(ctx, opts); err != nil {
				resp.OK = false
				resp.Error = err.Error()
			}
		}
	case "down":
		if err := a.Down(ctx); err != nil {
			resp.OK = false
			resp.Error = err.Error()
		}
	case "logout":
		if err := a.Logout(ctx); err != nil {
			resp.OK = false
			resp.Error = err.Error()
		}
	case "ping":
		if len(cmd.Payload) == 0 {
			resp.OK = false
			resp.Error = "missing target"
			break
		}
		var target string
		_ = json.Unmarshal(cmd.Payload, &target)
		out, err := exec.Command("ping", "-c", "3", "-W", "1", target).CombinedOutput()
		if err != nil {
			resp.OK = false
			resp.Error = err.Error()
		}
		resp.Data, _ = json.Marshal(string(out))
	default:
		resp.OK = false
		resp.Error = "unknown op: " + cmd.Op
	}
	_ = enc.Encode(resp)
}
