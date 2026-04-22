// trynet-derp is the relay. Clients open a WebSocket, announce their node
// key, and can send frames to any other connected peer by key.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/apolzek/trynet/internal/derp"
)

func main() {
	var (
		addr     = flag.String("addr", ":3478", "listen address")
		certFile = flag.String("cert", "", "TLS cert (optional)")
		keyFile  = flag.String("key", "", "TLS key (optional)")
	)
	flag.Parse()

	lg := log.New(os.Stderr, "derp ", log.LstdFlags|log.Lmsgprefix)
	srv := derp.New(lg)

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
