package main

import (
	"context"
	"fmt"
	"time"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	real := NewExternalQuoteAPI(map[string]float64{
		"PETR4": 38.42,
		"VALE3": 62.10,
	})
	real.Latency = 5 * time.Millisecond

	var api QuoteAPI = NewCachingRateLimitedProxy(real, 500*time.Millisecond, 3, 3)

	for i := 0; i < 5; i++ {
		q, err := api.Get(ctx, "PETR4")
		if err != nil {
			fmt.Println("erro:", err)
			continue
		}
		fmt.Printf("%s R$%.2f\n", q.Symbol, q.Price)
	}

	fmt.Printf("chamadas upstream: %d\n", real.Calls.Load())
}
