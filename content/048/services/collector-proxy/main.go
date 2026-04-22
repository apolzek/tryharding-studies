// collector-proxy runs as a sidecar next to the OpenTelemetry Collector.
// It validates the tenant ingest JWT on every OTLP-HTTP request and proxies
// valid ones to the local collector. Expired / wrong-tenant tokens → 401.
//
// Why not the collector's auth extension? The extension surface is coarse
// (static bearer or OIDC) and can't easily enforce a `tid` claim. A tiny
// reverse proxy is clearer, unit-testable, and upgradeable independently.
//
// gRPC ingest: for the POC we expose OTLP-HTTP on the public port. gRPC can
// be added by plugging a UnaryInterceptor that reads the `authorization`
// metadata — kept out for surface area.
package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/obs-saas/shared/config"
	jwtpkg "github.com/obs-saas/shared/jwt"
	"github.com/obs-saas/shared/log"
)

func main() {
	logger := log.New("collector-proxy")

	listen := config.Get("PROXY_LISTEN", ":8443")
	upstreamRaw := config.Must("UPSTREAM_HTTP")
	secret := []byte(config.Must("JWT_SECRET"))
	tid := config.Must("EXPECTED_TID")

	upstream, err := url.Parse(upstreamRaw)
	if err != nil {
		logger.Error("bad upstream", "err", err)
		os.Exit(1)
	}

	h := newHandler(logger, upstream, secret, tid)

	srv := &http.Server{
		Addr:              listen,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("listening", "addr", listen, "upstream", upstream.String(), "tid", tid)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

func newHandler(logger logger, upstream *url.URL, secret []byte, expectedTid string) http.Handler {
	rp := httputil.NewSingleHostReverseProxy(upstream)
	rp.ErrorLog = nil
	rp.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		logger.Error("upstream", "err", err, "path", r.URL.Path)
		http.Error(w, "bad gateway", http.StatusBadGateway)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("/v1/", authMW(logger, secret, expectedTid, rp))  // /v1/traces, /v1/metrics, /v1/logs
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	})
	return mux
}

type logger interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

func authMW(logger logger, secret []byte, expectedTid string, next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tok := extractBearer(r.Header.Get("Authorization"))
		if tok == "" {
			http.Error(w, "missing bearer token", http.StatusUnauthorized)
			return
		}
		claims, err := jwtpkg.Verify(secret, expectedTid, tok)
		if err != nil {
			logger.Warn("auth reject", "path", r.URL.Path, "err", err)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		// Strip caller-supplied headers before forwarding.
		r.Header.Del("Authorization")
		r.Header.Set("X-Tenant-Id", claims.TenantID)
		next.ServeHTTP(w, r)
	}
}

func extractBearer(h string) string {
	const p = "Bearer "
	if len(h) > len(p) && strings.EqualFold(h[:len(p)], p) {
		return strings.TrimSpace(h[len(p):])
	}
	return ""
}
