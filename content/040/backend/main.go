// userwatch backend: ingests eBPF-agent samples, persists them in SQLite,
// exposes a REST API and serves the static frontend.
//
// Designed to run as a single binary with no CGO dependencies.
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	addr := flag.String("addr", envOr("UW_ADDR", ":8080"), "listen address")
	dbPath := flag.String("db", envOr("UW_DB", "userwatch.db"), "sqlite database path")
	token := flag.String("token", envOr("UW_TOKEN", "dev-token"), "bearer token for /api/v1/ingest")
	frontend := flag.String("frontend", envOr("UW_FRONTEND", "./frontend"), "path to static frontend")
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	store, err := NewStore(*dbPath)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer store.Close()

	api := &API{store: store, token: *token}
	mux := http.NewServeMux()
	api.Register(mux)

	// Static frontend (SPA-ish: serve index.html on unknown paths).
	fs := http.FileServer(http.Dir(*frontend))
	mux.Handle("/", spaHandler(*frontend, fs))

	srv := &http.Server{
		Addr:              *addr,
		Handler:           withAccessLog(withCORS(mux)),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Background: roll sessions that have been idle too long.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sessionReaper(ctx, store)

	log.Printf("userwatch backend listening on %s (db=%s frontend=%s)",
		*addr, *dbPath, *frontend)

	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("serve: %v", err)
		}
	case <-sigs:
		log.Printf("shutdown requested")
		shutdownCtx, sc := context.WithTimeout(context.Background(), 5*time.Second)
		defer sc()
		_ = srv.Shutdown(shutdownCtx)
	}
}

// spaHandler serves files from `root`, falling back to index.html for unknown
// (non-api) paths so a client-side router can own routing. API paths are
// handled separately by the mux and never reach here.
func spaHandler(root string, fs http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		path := root + r.URL.Path
		if info, err := os.Stat(path); err != nil || info.IsDir() {
			// Fall through to index.html
			http.ServeFile(w, r, root+"/index.html")
			return
		}
		fs.ServeHTTP(w, r)
	})
}

func withCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(c int) {
	s.status = c
	s.ResponseWriter.WriteHeader(c)
}

func withAccessLog(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: 200}
		h.ServeHTTP(rec, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, rec.status, time.Since(start))
	})
}

// sessionReaper closes sessions that haven't seen samples in a while. A
// session is defined by (machine_id, session_user); we cut a new one after
// IdleGap of silence.
func sessionReaper(ctx context.Context, s *Store) {
	t := time.NewTicker(1 * time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := s.CloseIdleSessions(time.Now()); err != nil {
				log.Printf("session reaper: %v", err)
			}
		}
	}
}
