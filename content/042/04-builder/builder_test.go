package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newEchoServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		resp := map[string]any{
			"method":  r.Method,
			"path":    r.URL.Path,
			"query":   r.URL.RawQuery,
			"auth":    r.Header.Get("Authorization"),
			"ctype":   r.Header.Get("Content-Type"),
			"reqid":   r.Header.Get("X-Request-ID"),
			"bodyStr": strings.TrimSpace(string(body)),
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

func TestBuildValidation(t *testing.T) {
	cases := []struct {
		name    string
		mut     func(*RequestBuilder)
		wantErr error
	}{
		{"sem base url", func(b *RequestBuilder) { b.Method(http.MethodGet) }, ErrNoBaseURL},
		{"sem método", func(b *RequestBuilder) { b.BaseURL("http://x") }, ErrNoMethod},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := NewRequest(nil)
			tc.mut(b)
			if _, err := b.Build(context.Background()); !errors.Is(err, tc.wantErr) {
				t.Fatalf("esperado %v, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestFluentChainAgainstHTTPTest(t *testing.T) {
	srv := newEchoServer(t)
	defer srv.Close()

	status, body, err := NewRequest(srv.Client()).
		Method(http.MethodPost).
		BaseURL(srv.URL).
		Path("/v1/users").
		BearerAuth("abc").
		Header("X-Request-ID", "r-42").
		Query("flag", "true").
		Timeout(2 * time.Second).
		JSON(map[string]string{"name": "Ana"}).
		Do(context.Background())

	if err != nil {
		t.Fatal(err)
	}
	if status != http.StatusOK {
		t.Fatalf("status %d", status)
	}
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatal(err)
	}
	if got["method"] != "POST" {
		t.Fatalf("method esperado POST, got %v", got["method"])
	}
	if got["path"] != "/v1/users" {
		t.Fatalf("path esperado /v1/users, got %v", got["path"])
	}
	if got["auth"] != "Bearer abc" {
		t.Fatalf("auth esperado Bearer abc, got %v", got["auth"])
	}
	if got["reqid"] != "r-42" {
		t.Fatalf("reqid esperado r-42, got %v", got["reqid"])
	}
	if got["query"] != "flag=true" {
		t.Fatalf("query esperado flag=true, got %v", got["query"])
	}
	if got["ctype"] != "application/json" {
		t.Fatalf("ctype esperado application/json, got %v", got["ctype"])
	}
	if !strings.Contains(got["bodyStr"].(string), "\"name\":\"Ana\"") {
		t.Fatalf("body inesperado: %v", got["bodyStr"])
	}
}

func TestTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	_, _, err := NewRequest(srv.Client()).
		Method(http.MethodGet).
		BaseURL(srv.URL).
		Timeout(20 * time.Millisecond).
		Do(context.Background())
	if err == nil {
		t.Fatal("esperado timeout")
	}
}

func TestBuildWithoutBodyStillWorks(t *testing.T) {
	srv := newEchoServer(t)
	defer srv.Close()

	status, _, err := NewRequest(srv.Client()).
		Method(http.MethodGet).
		BaseURL(srv.URL).
		Path("/ping").
		Do(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if status != http.StatusOK {
		t.Fatalf("status %d", status)
	}
}

func TestInvalidJSONBody(t *testing.T) {
	// canais não são serializáveis em JSON, provoca erro no encode.
	_, err := NewRequest(nil).
		Method(http.MethodPost).
		BaseURL("http://x").
		JSON(make(chan int)).
		Build(context.Background())
	if err == nil {
		t.Fatal("esperado erro de encode")
	}
}
