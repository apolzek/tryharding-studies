// trynet-control is the coordination plane. It runs on a VPS, speaks HTTP to
// clients and to the admin UI, and persists state to a JSON file.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/apolzek/trynet/internal/control"
	"github.com/apolzek/trynet/internal/crypto"
)

func main() {
	var (
		addr       = flag.String("addr", ":8443", "listen address")
		statePath  = flag.String("state", "/var/lib/trynet/state.json", "path to state file")
		certFile   = flag.String("cert", "", "TLS cert (optional)")
		keyFile    = flag.String("key", "", "TLS key (optional)")
		adminToken = flag.String("admin-token", os.Getenv("TRYNET_ADMIN_TOKEN"), "admin token for UI access")
		derpURL    = flag.String("derp-url", os.Getenv("TRYNET_DERP_URL"), "public URL of the relay")
	)
	flag.Parse()

	lg := log.New(os.Stderr, "control ", log.LstdFlags|log.Lmsgprefix)

	store, err := control.NewStore(*statePath)
	if err != nil {
		lg.Fatalf("init store: %v", err)
	}

	if err := store.UpdateSettings(func(cur *control.Settings) {
		if *adminToken != "" {
			cur.AdminToken = *adminToken
		}
		if cur.AdminToken == "" {
			cur.AdminToken = crypto.NewToken(24)
			lg.Printf("generated admin token: %s", cur.AdminToken)
		}
		if *derpURL != "" {
			cur.DerpURL = *derpURL
		}
	}); err != nil {
		lg.Fatalf("settings: %v", err)
	}

	srv := control.New(store, lg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		cancel()
	}()

	if err := srv.Run(ctx, *addr, *certFile, *keyFile); err != nil {
		lg.Fatalf("run: %v", err)
	}
}
