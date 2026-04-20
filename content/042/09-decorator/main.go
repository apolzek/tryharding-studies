package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
)

func main() {
	logger := log.New(os.Stdout, "http: ", log.LstdFlags)
	run(logger, os.Stdout)
}

func run(logger *log.Logger, w io.Writer) {
	metrics := &Metrics{}
	limiter := NewRateLimiter(3, 3)

	base := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusOK)
		_, _ = rw.Write([]byte("ok"))
	})

	handler := Chain(base,
		Logging(logger),
		MetricsMiddleware(metrics),
		RateLimit(limiter),
		Auth("s3cr3t"),
	)

	srv := httptest.NewServer(handler)
	defer srv.Close()

	resp, _ := http.Get(srv.URL + "/ping")
	fmt.Fprintln(w, "unauth status:", resp.StatusCode)
	resp.Body.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/ping", nil)
	req.Header.Set("Authorization", "Bearer s3cr3t")
	resp, _ = http.DefaultClient.Do(req)
	fmt.Fprintln(w, "auth status:", resp.StatusCode)
	resp.Body.Close()

	for i := 0; i < 5; i++ {
		req, _ := http.NewRequest("GET", srv.URL+"/ping", nil)
		req.Header.Set("Authorization", "Bearer s3cr3t")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Fprintln(w, "err:", err)
			continue
		}
		fmt.Fprintf(w, "burst[%d]=%d\n", i, r.StatusCode)
		r.Body.Close()
	}

	fmt.Fprintf(w, "metrics: requests=%d errors=%d\n",
		metrics.Requests.Load(), metrics.Errors.Load())
}
