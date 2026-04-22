// trynetd is the on-device daemon. It reads /etc/trynet/config.json, keeps a
// connection to the control server, and drives WireGuard on the local box.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/apolzek/trynet/internal/client"
)

func main() {
	var (
		cfgPath   = flag.String("config", "/etc/trynet/config.json", "daemon config")
		statePath = flag.String("state", "/var/lib/trynet/client.json", "persistent state")
		sockPath  = flag.String("sock", "/run/trynetd.sock", "CLI socket")
		ifName    = flag.String("iface", "wg0", "wireguard interface")
	)
	flag.Parse()

	lg := log.New(os.Stderr, "trynetd ", log.LstdFlags|log.Lmsgprefix)

	cfg, err := client.LoadConfig(*cfgPath)
	if err != nil {
		lg.Fatalf("config: %v", err)
	}
	agent, err := client.NewAgent(cfg, *statePath, *ifName, lg)
	if err != nil {
		lg.Fatalf("agent: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		cancel()
	}()

	// Start the netmap loop if we already have state.
	go func() { _ = agent.Start(ctx) }()

	if err := agent.ServeLocal(ctx, *sockPath); err != nil {
		lg.Fatalf("local: %v", err)
	}
}
