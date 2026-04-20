package main

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("pong"))
	})
}

func TestAuthMiddleware(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		header  string
		want    int
		wantMsg string
	}{
		{"no header", "", http.StatusUnauthorized, "unauthorized"},
		{"wrong token", "Bearer nope", http.StatusUnauthorized, "unauthorized"},
		{"missing bearer prefix", "abc", http.StatusUnauthorized, "unauthorized"},
		{"correct token", "Bearer tok", http.StatusOK, "pong"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := Chain(okHandler(), Auth("tok"))
			req := httptest.NewRequest("GET", "/", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)
			if w.Code != tc.want {
				t.Fatalf("status=%d want %d", w.Code, tc.want)
			}
			if !strings.Contains(w.Body.String(), tc.wantMsg) {
				t.Fatalf("body=%q want substring %q", w.Body.String(), tc.wantMsg)
			}
		})
	}
}

func TestRateLimit(t *testing.T) {
	t.Parallel()
	// 2 tokens, recuperação lenta para forçar bloqueio no 3.
	limiter := NewRateLimiter(2, 0.0001)
	h := Chain(okHandler(), RateLimit(limiter))

	for i, want := range []int{http.StatusOK, http.StatusOK, http.StatusTooManyRequests} {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != want {
			t.Errorf("call %d status=%d want %d", i, w.Code, want)
		}
	}
}

func TestRateLimit_Refill(t *testing.T) {
	t.Parallel()
	limiter := NewRateLimiter(1, 1)
	fake := time.Unix(0, 0)
	limiter.now = func() time.Time { return fake }
	limiter.lastRefill = fake

	if !limiter.Allow() {
		t.Fatalf("first should pass")
	}
	if limiter.Allow() {
		t.Fatalf("second should fail (no refill yet)")
	}
	fake = fake.Add(2 * time.Second)
	if !limiter.Allow() {
		t.Fatalf("after refill should pass")
	}
}

func TestLoggingMiddlewareWrites(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)
	h := Chain(okHandler(), Logging(logger))

	req := httptest.NewRequest("GET", "/foo", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	out := buf.String()
	if !strings.Contains(out, "GET /foo") {
		t.Errorf("log missing path: %q", out)
	}
	if !strings.Contains(out, "200") {
		t.Errorf("log missing status: %q", out)
	}
}

func TestMetricsMiddleware(t *testing.T) {
	t.Parallel()
	m := &Metrics{}
	errH := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})
	h := Chain(errH, MetricsMiddleware(m))

	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	}
	if got := m.Requests.Load(); got != 3 {
		t.Errorf("requests=%d want 3", got)
	}
	if got := m.Errors.Load(); got != 3 {
		t.Errorf("errors=%d want 3", got)
	}
}

func TestChainOrder(t *testing.T) {
	t.Parallel()
	var trace []string
	mk := func(name string) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				trace = append(trace, "pre "+name)
				next.ServeHTTP(w, r)
				trace = append(trace, "post "+name)
			})
		}
	}
	h := Chain(okHandler(), mk("A"), mk("B"))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))

	want := []string{"pre A", "pre B", "post B", "post A"}
	if strings.Join(trace, ",") != strings.Join(want, ",") {
		t.Fatalf("order=%v want %v", trace, want)
	}
}

func TestRunDemoOutputs(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	run(DiscardLogger(), &buf)
	out := buf.String()
	for _, want := range []string{"unauth status:", "auth status:", "burst[", "metrics:"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q: %s", want, out)
		}
	}
}

func TestFullStack_Integration(t *testing.T) {
	t.Parallel()
	metrics := &Metrics{}
	limiter := NewRateLimiter(5, 5)
	handler := Chain(okHandler(),
		Logging(DiscardLogger()),
		MetricsMiddleware(metrics),
		RateLimit(limiter),
		Auth("tok"),
	)
	srv := httptest.NewServer(handler)
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL, nil)
	req.Header.Set("Authorization", "Bearer tok")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if got := metrics.Requests.Load(); got != 1 {
		t.Errorf("requests=%d want 1", got)
	}
}
