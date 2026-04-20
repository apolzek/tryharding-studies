package main

import (
	"context"
	"fmt"
	"time"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	pipeline := BuildPipeline(
		map[string]string{"tok-123": "alice"},
		3,
		[]string{"amount", "currency"},
		1000.0,
	)

	reqs := []*Request{
		{Token: "tok-123", ClientIP: "10.0.0.1", Path: "/pay", Body: map[string]any{"amount": 50.0, "currency": "BRL"}},
		{Token: "tok-bad", ClientIP: "10.0.0.2", Path: "/pay", Body: map[string]any{"amount": 10.0, "currency": "BRL"}},
		{Token: "tok-123", ClientIP: "10.0.0.1", Path: "/pay", Body: map[string]any{"amount": 5000.0, "currency": "BRL"}},
		{Token: "tok-123", ClientIP: "10.0.0.1", Path: "/pay", Body: map[string]any{"currency": "BRL"}},
	}

	for i, r := range reqs {
		err := pipeline.Handle(ctx, r)
		if err != nil {
			fmt.Printf("req[%d] rejeitada: %v\n", i, err)
			continue
		}
		fmt.Printf("req[%d] aceita para user=%s\n", i, r.User)
	}
}
