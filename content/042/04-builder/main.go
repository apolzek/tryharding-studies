package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"
)

func main() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("[server] %s %s auth=%q ct=%q q=%s\n",
			r.Method, r.URL.Path, r.Header.Get("Authorization"),
			r.Header.Get("Content-Type"), r.URL.RawQuery)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "path": r.URL.Path})
	}))
	defer srv.Close()

	ctx := context.Background()
	status, body, err := NewRequest(srv.Client()).
		Method(http.MethodPost).
		BaseURL(srv.URL).
		Path("/v1/orders").
		BearerAuth("token-xyz").
		Header("X-Request-ID", "abc-123").
		Query("ref", "promo").
		Timeout(2 * time.Second).
		JSON(map[string]any{"sku": "abc", "qty": 2}).
		Do(ctx)
	if err != nil {
		fmt.Println("erro:", err)
		return
	}
	fmt.Printf("[client] status=%d body=%s\n", status, body)
}
