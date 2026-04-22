// trynet-ui is the admin web console. It renders HTML and proxies mutations
// to the control server using the shared admin token.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/apolzek/trynet/internal/ui"
)

func main() {
	var (
		addr       = flag.String("addr", ":8080", "listen address")
		controlURL = flag.String("control-url", envOr("TRYNET_CONTROL_URL", "http://trynet-control:8443"), "control server URL")
		adminToken = flag.String("admin-token", os.Getenv("TRYNET_ADMIN_TOKEN"), "shared secret")
		insecure   = flag.Bool("insecure", false, "skip TLS verification when talking to control")
		user       = flag.String("user", os.Getenv("TRYNET_UI_USER"), "basic-auth user (optional)")
		pass       = flag.String("pass", os.Getenv("TRYNET_UI_PASS"), "basic-auth password (optional)")
	)
	flag.Parse()

	lg := log.New(os.Stderr, "ui ", log.LstdFlags|log.Lmsgprefix)

	srv, err := ui.New(ui.Config{
		Addr:       *addr,
		ControlURL: *controlURL,
		AdminToken: *adminToken,
		Insecure:   *insecure,
		UIUser:     *user,
		UIPass:     *pass,
	}, lg)
	if err != nil {
		lg.Fatalf("ui: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		cancel()
	}()

	if err := srv.Run(ctx); err != nil {
		lg.Fatalf("run: %v", err)
	}
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
