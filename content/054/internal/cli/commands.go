// Package cli implements the small `trynet` command-line tool that talks to
// the local daemon over a UNIX socket.
package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/apolzek/trynet/internal/client"
	"github.com/apolzek/trynet/internal/protocol"
)

// Run dispatches argv to a subcommand and returns an exit code.
func Run(args []string) int {
	if len(args) < 1 {
		usage(os.Stderr)
		return 2
	}
	sub := args[0]
	switch sub {
	case "up":
		return cmdUp(args[1:])
	case "down":
		return cmdDown(args[1:])
	case "logout":
		return cmdLogout(args[1:])
	case "status":
		return cmdStatus(args[1:])
	case "ip":
		return cmdIP(args[1:])
	case "ping":
		return cmdPing(args[1:])
	case "version":
		fmt.Println("trynet 0.1.0")
		return 0
	case "help", "-h", "--help":
		usage(os.Stdout)
		return 0
	default:
		usage(os.Stderr)
		return 2
	}
}

func usage(w *os.File) {
	fmt.Fprintln(w, `Usage: trynet <command> [flags]

Commands:
  up        Activate the tunnel (requires --authkey the first time)
  down      Disable the tunnel but keep credentials
  logout    Disable and forget credentials
  status    Show daemon state and peers
  ip        Print this node's tailnet IP
  ping      Ping a peer by MagicDNS name or IP (uses daemon)
  version   Print version`)
}

// -----------------------------------------------------------------------------
// commands
// -----------------------------------------------------------------------------

func cmdUp(args []string) int {
	fs := flag.NewFlagSet("up", flag.ExitOnError)
	authKey := fs.String("authkey", "", "pre-auth key from the admin UI")
	hostname := fs.String("hostname", "", "override hostname")
	routes := fs.String("advertise-routes", "", "comma-separated CIDRs to advertise")
	exitNode := fs.Bool("exit-node", false, "advertise as exit node")
	_ = fs.Parse(args)

	opts := client.LoginOptions{
		AuthKey:  *authKey,
		Hostname: *hostname,
		ExitNode: *exitNode,
	}
	if *routes != "" {
		opts.Routes = strings.Split(*routes, ",")
	}
	payload, _ := json.Marshal(opts)
	resp, err := send(protocol.LocalCommand{Op: "up", Payload: payload})
	if err != nil {
		fmt.Fprintln(os.Stderr, "up:", err)
		return 1
	}
	if !resp.OK {
		fmt.Fprintln(os.Stderr, "up:", resp.Error)
		return 1
	}
	fmt.Println("up: requested; run `trynet status` to verify")
	return 0
}

func cmdDown(args []string) int {
	resp, err := send(protocol.LocalCommand{Op: "down"})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if !resp.OK {
		fmt.Fprintln(os.Stderr, resp.Error)
		return 1
	}
	return 0
}

func cmdLogout(args []string) int {
	resp, err := send(protocol.LocalCommand{Op: "logout"})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if !resp.OK {
		fmt.Fprintln(os.Stderr, resp.Error)
		return 1
	}
	fmt.Println("logged out")
	return 0
}

func cmdStatus(args []string) int {
	resp, err := send(protocol.LocalCommand{Op: "status"})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if !resp.OK {
		fmt.Fprintln(os.Stderr, resp.Error)
		return 1
	}
	var st protocol.LocalStatus
	if err := json.Unmarshal(resp.Data, &st); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Print(client.PrettyStatus(st))
	return 0
}

func cmdIP(args []string) int {
	resp, err := send(protocol.LocalCommand{Op: "status"})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	var st protocol.LocalStatus
	_ = json.Unmarshal(resp.Data, &st)
	if !st.TailnetIP.IsValid() {
		fmt.Fprintln(os.Stderr, "no tailnet IP; not logged in?")
		return 1
	}
	fmt.Println(st.TailnetIP)
	return 0
}

func cmdPing(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: trynet ping <host-or-ip>")
		return 2
	}
	target, _ := json.Marshal(args[0])
	resp, err := send(protocol.LocalCommand{Op: "ping", Payload: target})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if !resp.OK {
		fmt.Fprintln(os.Stderr, resp.Error)
	}
	var out string
	_ = json.Unmarshal(resp.Data, &out)
	fmt.Print(out)
	if !resp.OK {
		return 1
	}
	return 0
}

// -----------------------------------------------------------------------------
// socket helper
// -----------------------------------------------------------------------------

func sockPath() string {
	if p := os.Getenv("TRYNETD_SOCK"); p != "" {
		return p
	}
	return "/run/trynetd.sock"
}

func send(cmd protocol.LocalCommand) (*protocol.LocalResponse, error) {
	c, err := net.Dial("unix", sockPath())
	if err != nil {
		return nil, fmt.Errorf("dial daemon: %w", err)
	}
	defer c.Close()
	if err := json.NewEncoder(c).Encode(&cmd); err != nil {
		return nil, err
	}
	var resp protocol.LocalResponse
	if err := json.NewDecoder(c).Decode(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
