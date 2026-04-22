package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiterAllowsWithinQuota(t *testing.T) {
	rl := newRateLimiter(3, time.Minute)
	now := time.Now()
	for i := 0; i < 3; i++ {
		if !rl.allow("1.2.3.4", now) {
			t.Fatalf("req %d should pass", i)
		}
	}
	if rl.allow("1.2.3.4", now) {
		t.Fatal("4th should be blocked")
	}
}

func TestRateLimiterIsolatesByIP(t *testing.T) {
	rl := newRateLimiter(1, time.Minute)
	now := time.Now()
	if !rl.allow("a", now) || !rl.allow("b", now) {
		t.Fatal("different IPs should each get their own quota")
	}
	if rl.allow("a", now) {
		t.Fatal("a should be blocked")
	}
}

func TestRateLimiterExpiresOldHits(t *testing.T) {
	rl := newRateLimiter(1, 10*time.Millisecond)
	now := time.Now()
	if !rl.allow("x", now) {
		t.Fatal("first should pass")
	}
	if !rl.allow("x", now.Add(20*time.Millisecond)) {
		t.Fatal("after window, should pass again")
	}
}

func TestClientIPForwardedFor(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Forwarded-For", "1.1.1.1, 2.2.2.2")
	if got := clientIP(r); got != "1.1.1.1" {
		t.Fatalf("got %q", got)
	}
}

func TestClientIPRemoteAddr(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "10.0.0.1:5555"
	if got := clientIP(r); got != "10.0.0.1" {
		t.Fatalf("got %q", got)
	}
}

func TestCORSPreflight(t *testing.T) {
	h := cors(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Fatal("next should not run on preflight") }))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("OPTIONS", "/anything", nil))
	if rr.Code != 204 {
		t.Fatalf("code %d", rr.Code)
	}
	if rr.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatal("missing CORS header")
	}
}
