package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"sync/atomic"
	"time"
)

var (
	received int64
	secret   = envOr("WEBHOOK_SECRET", "s3cret")
)

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func verifyHMAC(body []byte, sig string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(sig))
}

func webhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}
	sig := r.Header.Get("X-Signature")
	if sig != "" && !verifyHMAC(body, sig) {
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}
	var payload map[string]any
	_ = json.Unmarshal(body, &payload)
	n := atomic.AddInt64(&received, 1)
	log.Printf("[go-wh] #%d event=%v", n, payload["event"])
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(`{"status":"accepted"}`))
}

func stats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"received": atomic.LoadInt64(&received),
		"ts":       time.Now().Unix(),
	})
}

func health(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) }

func main() {
	http.HandleFunc("/webhook", webhook)
	http.HandleFunc("/stats", stats)
	http.HandleFunc("/health", health)
	addr := ":9001"
	log.Printf("[go-wh] listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
