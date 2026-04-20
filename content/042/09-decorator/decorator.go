package main

import (
	"io"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// Middleware é uma função que decora um http.Handler.
type Middleware func(http.Handler) http.Handler

// Chain aplica os middlewares na ordem dada. O primeiro da lista fica no
// topo da cadeia (mais externo).
func Chain(h http.Handler, mws ...Middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

// responseRecorder captura o status e bytes escritos para métricas/log.
type responseRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *responseRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

// Logging escreve uma linha por requisição.
func Logging(logger *log.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &responseRecorder{ResponseWriter: w}
			next.ServeHTTP(rec, r)
			logger.Printf("%s %s -> %d (%d bytes) %s",
				r.Method, r.URL.Path, rec.status, rec.bytes, time.Since(start))
		})
	}
}

// Auth valida um header Authorization "Bearer <token>".
func Auth(expected string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got := r.Header.Get("Authorization")
			if got != "Bearer "+expected {
				w.Header().Set("WWW-Authenticate", "Bearer")
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimiter implementa um token-bucket simples global (por processo).
type RateLimiter struct {
	mu         sync.Mutex
	tokens     float64
	max        float64
	refillRate float64 // tokens por segundo
	lastRefill time.Time
	now        func() time.Time
}

func NewRateLimiter(max, refillPerSecond float64) *RateLimiter {
	return &RateLimiter{
		tokens:     max,
		max:        max,
		refillRate: refillPerSecond,
		lastRefill: time.Now(),
		now:        time.Now,
	}
}

func (l *RateLimiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	elapsed := now.Sub(l.lastRefill).Seconds()
	l.tokens = minFloat(l.max, l.tokens+elapsed*l.refillRate)
	l.lastRefill = now
	if l.tokens >= 1 {
		l.tokens--
		return true
	}
	return false
}

// RateLimit middleware bloqueia quando excede a taxa.
func RateLimit(l *RateLimiter) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !l.Allow() {
				w.Header().Set("Retry-After", "1")
				http.Error(w, "rate limited", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Metrics contador simples thread-safe.
type Metrics struct {
	Requests atomic.Int64
	Errors   atomic.Int64
}

// MetricsMiddleware conta requests e erros (status >= 500).
func MetricsMiddleware(m *Metrics) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			m.Requests.Add(1)
			rec := &responseRecorder{ResponseWriter: w}
			next.ServeHTTP(rec, r)
			if rec.status >= 500 {
				m.Errors.Add(1)
			}
		})
	}
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// DiscardLogger devolve um logger que não escreve em nada (útil em testes e demos).
func DiscardLogger() *log.Logger {
	return log.New(io.Discard, "", 0)
}
