package main

import (
	"context"
	"runtime"
	"testing"
	"time"
)

func TestPipeline(t *testing.T) {
	t.Run("pipeline completo processa válidos e descarta inválidos", func(t *testing.T) {
		ctx := context.Background()
		raw := []string{
			"alice, buy, 50",
			"bob, buy, 250",
			"ruim",
			"carol, sell, 1200",
			"x, y, abc",
		}
		out := Persist(ctx, Enrich(ctx, Parse(ctx, Ingest(ctx, raw))))

		var got []Persisted
		for r := range out {
			got = append(got, r)
		}
		if len(got) != 3 {
			t.Fatalf("esperava 3 registros, obteve %d", len(got))
		}
		if got[0].Tier != "bronze" || got[1].Tier != "silver" || got[2].Tier != "gold" {
			t.Fatalf("tiers incorretos: %+v", got)
		}
		if got[2].ID != 3 {
			t.Fatalf("IDs não sequenciais: %+v", got)
		}
	})

	t.Run("cancelamento via context encerra sem vazar", func(t *testing.T) {
		before := runtime.NumGoroutine()
		ctx, cancel := context.WithCancel(context.Background())

		big := make([]string, 10000)
		for i := range big {
			big[i] = "u, a, 1"
		}
		out := Persist(ctx, Enrich(ctx, Parse(ctx, Ingest(ctx, big))))

		// Consome só algumas e cancela.
		for i := 0; i < 5; i++ {
			<-out
		}
		cancel()
		// drena
		for range out {
		}

		time.Sleep(50 * time.Millisecond)
		after := runtime.NumGoroutine()
		if after > before+2 {
			t.Fatalf("vazamento de goroutines: antes=%d depois=%d", before, after)
		}
	})

	t.Run("entrada vazia fecha todos os canais", func(t *testing.T) {
		ctx := context.Background()
		out := Persist(ctx, Enrich(ctx, Parse(ctx, Ingest(ctx, nil))))
		var got int
		for range out {
			got++
		}
		if got != 0 {
			t.Fatalf("esperava 0 registros, obteve %d", got)
		}
	})

	t.Run("format produz string estável", func(t *testing.T) {
		p := Persisted{ID: 7, Enriched: Enriched{Parsed: Parsed{User: "ana", Action: "buy", Amount: 10}, Tier: "bronze"}}
		got := Format(p)
		want := "id=7 user=ana action=buy amount=10 tier=bronze"
		if got != want {
			t.Fatalf("esperava %q, obteve %q", want, got)
		}
	})
}
