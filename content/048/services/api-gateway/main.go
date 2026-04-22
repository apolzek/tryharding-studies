// api-gateway — single browser-facing entrypoint. Proxies to auth-service.
// Adds CORS, basic rate-limit, and request logging. Stateless.
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
	"sync"
	"syscall"
	"time"

	"github.com/obs-saas/shared/config"
	"github.com/obs-saas/shared/log"
)

func main() {
	logger := log.New("api-gateway")
	authURL, err := url.Parse(config.Get("OBS_AUTH_URL", "http://auth:8081"))
	if err != nil {
		logger.Error("bad OBS_AUTH_URL", "err", err)
		os.Exit(1)
	}
	rp := httputil.NewSingleHostReverseProxy(authURL)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	mux.Handle("/api/v1/register", rewrite("/register", rp))
	mux.Handle("/api/v1/login", rewrite("/login", rp))
	mux.Handle("/api/v1/tenants/", rewriteStrip("/api/v1", rp))

	rl := newRateLimiter(20, time.Minute) // 20 req/min per IP
	h := cors(rl.wrap(logging(logger, mux)))

	srv := &http.Server{
		Addr:              config.Get("OBS_LISTEN", ":8080"),
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		logger.Info("listening", "addr", srv.Addr, "auth", authURL.String())
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

func rewrite(toPath string, rp http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = toPath
		rp.ServeHTTP(w, r)
	})
}

func rewriteStrip(prefix string, rp http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = strings.TrimPrefix(r.URL.Path, prefix)
		rp.ServeHTTP(w, r)
	})
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("access-control-allow-origin", "*")
		w.Header().Set("access-control-allow-headers", "content-type,authorization")
		w.Header().Set("access-control-allow-methods", "GET,POST,OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func logging(lg interface{ Info(string, ...any) }, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRec{ResponseWriter: w, code: 200}
		next.ServeHTTP(rec, r)
		lg.Info("http", "method", r.Method, "path", r.URL.Path, "status", rec.code, "dur_ms", time.Since(start).Milliseconds())
	})
}

type statusRec struct {
	http.ResponseWriter
	code int
}

func (s *statusRec) WriteHeader(c int) { s.code = c; s.ResponseWriter.WriteHeader(c) }

// ---- rate limiter (token bucket per-IP, simple & test-covered) ----

type rateLimiter struct {
	mu     sync.Mutex
	quota  int
	window time.Duration
	hits   map[string][]time.Time
}

func newRateLimiter(quota int, window time.Duration) *rateLimiter {
	return &rateLimiter{quota: quota, window: window, hits: map[string][]time.Time{}}
}

func (rl *rateLimiter) allow(ip string, now time.Time) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	cut := now.Add(-rl.window)
	h := rl.hits[ip]
	i := 0
	for ; i < len(h); i++ {
		if h[i].After(cut) {
			break
		}
	}
	h = h[i:]
	if len(h) >= rl.quota {
		rl.hits[ip] = h
		return false
	}
	rl.hits[ip] = append(h, now)
	return true
}

func (rl *rateLimiter) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if !rl.allow(ip, time.Now()) {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func clientIP(r *http.Request) string {
	if v := r.Header.Get("X-Forwarded-For"); v != "" {
		if i := strings.IndexByte(v, ','); i >= 0 {
			return strings.TrimSpace(v[:i])
		}
		return strings.TrimSpace(v)
	}
	if i := strings.LastIndexByte(r.RemoteAddr, ':'); i >= 0 {
		return r.RemoteAddr[:i]
	}
	return r.RemoteAddr
}
