package main

import (
	"context"
	"fmt"
	"time"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	providers := []Provider{
		FakeProvider{ID: "broker-a", Delta: 0.12},
		FakeProvider{ID: "broker-b", Delta: -0.05},
		FakeProvider{ID: "broker-c", Delta: 0.30},
		FakeProvider{ID: "broker-d", Fail: true},
	}

	quotes := Aggregate(ctx, providers, "BTC")
	for _, q := range quotes {
		if q.Err != nil {
			fmt.Printf("provider=%s erro=%v\n", q.Provider, q.Err)
			continue
		}
		fmt.Printf("provider=%s symbol=%s price=%.2f\n", q.Provider, q.Symbol, q.Price)
	}

	best, err := BestPrice(quotes)
	if err != nil {
		fmt.Println("sem melhor preço:", err)
		return
	}
	fmt.Printf("melhor preço: %s @ %.2f\n", best.Provider, best.Price)
}
