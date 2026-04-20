package main

import (
	"context"
	"runtime"
	"testing"
	"time"
)

func TestFanOutFanIn(t *testing.T) {
	providers := []Provider{
		FakeProvider{ID: "a", Delta: 1},
		FakeProvider{ID: "b", Delta: 2},
		FakeProvider{ID: "c", Delta: 3},
	}

	t.Run("agrega todos os provedores", func(t *testing.T) {
		ctx := context.Background()
		qs := Aggregate(ctx, providers, "ACME")
		if len(qs) != 3 {
			t.Fatalf("esperava 3 quotes, obteve %d", len(qs))
		}
		seen := map[string]bool{}
		for _, q := range qs {
			if q.Err != nil {
				t.Fatalf("erro inesperado: %v", q.Err)
			}
			seen[q.Provider] = true
		}
		if len(seen) != 3 {
			t.Fatalf("provedores faltando: %+v", seen)
		}
	})

	t.Run("erros não abortam o agregado", func(t *testing.T) {
		ctx := context.Background()
		mix := []Provider{
			FakeProvider{ID: "ok1", Delta: 1},
			FakeProvider{ID: "bad", Fail: true},
			FakeProvider{ID: "ok2", Delta: 2},
		}
		qs := Aggregate(ctx, mix, "ACME")
		if len(qs) != 3 {
			t.Fatalf("esperava 3 quotes, obteve %d", len(qs))
		}
		var okCount, errCount int
		for _, q := range qs {
			if q.Err != nil {
				errCount++
			} else {
				okCount++
			}
		}
		if okCount != 2 || errCount != 1 {
			t.Fatalf("esperava 2 ok e 1 erro, obteve ok=%d err=%d", okCount, errCount)
		}
	})

	t.Run("best price escolhe menor", func(t *testing.T) {
		qs := []Quote{
			{Provider: "a", Price: 101},
			{Provider: "b", Price: 99.5},
			{Provider: "c", Price: 102},
			{Provider: "d", Err: context.Canceled},
		}
		best, err := BestPrice(qs)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if best.Provider != "b" {
			t.Fatalf("esperava b, obteve %s", best.Provider)
		}
	})

	t.Run("best price em tudo inválido retorna erro", func(t *testing.T) {
		qs := []Quote{{Err: context.Canceled}}
		if _, err := BestPrice(qs); err == nil {
			t.Fatal("esperava erro")
		}
	})

	t.Run("cancelamento não vaza goroutines", func(t *testing.T) {
		before := runtime.NumGoroutine()
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // cancela antes de iniciar
		qs := Aggregate(ctx, providers, "ACME")
		_ = qs
		time.Sleep(50 * time.Millisecond)
		after := runtime.NumGoroutine()
		if after > before+2 {
			t.Fatalf("vazamento: antes=%d depois=%d", before, after)
		}
	})

	t.Run("lista vazia retorna canal vazio", func(t *testing.T) {
		ctx := context.Background()
		qs := Aggregate(ctx, nil, "ACME")
		if len(qs) != 0 {
			t.Fatalf("esperava 0, obteve %d", len(qs))
		}
	})
}
