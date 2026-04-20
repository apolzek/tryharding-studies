package main

import (
	"context"
	"fmt"
	"time"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	raw := []string{
		"alice, buy, 50",
		"bob, buy, 250",
		"carol, sell, 1200",
		"linha-invalida",
		"dave, buy, abc",
		"erin, sell, 80",
	}

	// Estágios encadeados.
	events := Ingest(ctx, raw)
	parsed := Parse(ctx, events)
	enriched := Enrich(ctx, parsed)
	persisted := Persist(ctx, enriched)

	for rec := range persisted {
		fmt.Println(Format(rec))
	}
	fmt.Println("pipeline encerrado")
}
