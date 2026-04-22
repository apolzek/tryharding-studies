package main

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	jwtpkg "github.com/obs-saas/shared/jwt"
)

func newTestHandler(t *testing.T, upstream *httptest.Server, secret []byte, tid string) http.Handler {
	t.Helper()
	u, _ := url.Parse(upstream.URL)
	return newHandler(slog.New(slog.NewTextHandler(io.Discard, nil)), u, secret, tid)
}

func TestRejectsMissingToken(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer up.Close()
	h := newTestHandler(t, up, []byte("0123456789abcdef0123456789abcdef"), "t-abc")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("POST", "/v1/traces", strings.NewReader("{}")))
	if rr.Code != 401 {
		t.Fatalf("got %d", rr.Code)
	}
}

func TestRejectsBadToken(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer up.Close()
	h := newTestHandler(t, up, []byte("0123456789abcdef0123456789abcdef"), "t-abc")
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/v1/traces", strings.NewReader("{}"))
	req.Header.Set("Authorization", "Bearer not-a-token")
	h.ServeHTTP(rr, req)
	if rr.Code != 401 {
		t.Fatalf("got %d", rr.Code)
	}
}

func TestForwardsValidToken(t *testing.T) {
	secret := []byte("0123456789abcdef0123456789abcdef")
	tok, _ := jwtpkg.Issue(secret, "t-abc", time.Hour)
	var seenTid, seenAuth string
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenTid = r.Header.Get("X-Tenant-Id")
		seenAuth = r.Header.Get("Authorization")
		w.WriteHeader(202)
	}))
	defer up.Close()
	h := newTestHandler(t, up, secret, "t-abc")
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/v1/traces", strings.NewReader("{}"))
	req.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rr, req)
	if rr.Code != 202 {
		t.Fatalf("got %d body=%s", rr.Code, rr.Body.String())
	}
	if seenTid != "t-abc" {
		t.Fatalf("upstream saw tid = %q", seenTid)
	}
	if seenAuth != "" {
		t.Fatalf("upstream should not see bearer, got %q", seenAuth)
	}
}

func TestRejectsTidMismatch(t *testing.T) {
	secret := []byte("0123456789abcdef0123456789abcdef")
	tok, _ := jwtpkg.Issue(secret, "t-other", time.Hour)
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Fatal("forwarded") }))
	defer up.Close()
	h := newTestHandler(t, up, secret, "t-abc")
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/v1/traces", strings.NewReader("{}"))
	req.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rr, req)
	if rr.Code != 401 {
		t.Fatalf("got %d", rr.Code)
	}
}

func TestHealthz(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer up.Close()
	h := newTestHandler(t, up, []byte("0123456789abcdef0123456789abcdef"), "t-abc")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/healthz", nil))
	if rr.Code != 200 {
		t.Fatalf("got %d", rr.Code)
	}
}
