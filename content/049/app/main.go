package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strconv"
	"time"
)

type response struct {
	Endpoint   string  `json:"endpoint"`
	Complexity string  `json:"complexity"`
	N          int     `json:"n"`
	Result     float64 `json:"result"`
	ElapsedMs  float64 `json:"elapsed_ms"`
}

// simple: O(1) — constant time arithmetic.
func simpleHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	v := 1.0
	for i := 0; i < 10; i++ {
		v = v*1.0001 + 0.5
	}
	writeJSON(w, response{
		Endpoint:   "/simple",
		Complexity: "O(1)",
		N:          10,
		Result:     v,
		ElapsedMs:  float64(time.Since(start).Microseconds()) / 1000.0,
	})
}

// medium: O(n log n) — sort a random slice.
func mediumHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	n := parseN(r, 2000, 1, 200000)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	data := make([]float64, n)
	for i := range data {
		data[i] = rng.Float64()
	}
	sort.Float64s(data)
	writeJSON(w, response{
		Endpoint:   "/medium",
		Complexity: "O(n log n)",
		N:          n,
		Result:     data[n-1],
		ElapsedMs:  float64(time.Since(start).Microseconds()) / 1000.0,
	})
}

// heavy: O(2^n) — recursive Fibonacci, a textbook "Big-Complexity" endpoint.
func heavyHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	n := parseN(r, 25, 1, 35) // cap to avoid unbounded latency
	result := fib(n)
	writeJSON(w, response{
		Endpoint:   "/heavy",
		Complexity: "O(2^n)",
		N:          n,
		Result:     float64(result),
		ElapsedMs:  float64(time.Since(start).Microseconds()) / 1000.0,
	})
}

func fib(n int) uint64 {
	if n < 2 {
		return uint64(n)
	}
	return fib(n-1) + fib(n-2)
}

func parseN(r *http.Request, def, min, max int) int {
	q := r.URL.Query().Get("n")
	if q == "" {
		return def
	}
	v, err := strconv.Atoi(q)
	if err != nil {
		return def
	}
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func writeJSON(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(body)
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func main() {
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":8765"
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/simple", simpleHandler)
	mux.HandleFunc("/medium", mediumHandler)
	mux.HandleFunc("/heavy", heavyHandler)

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("bigbench listening on %s", addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
