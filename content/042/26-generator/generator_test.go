package main

import (
	"context"
	"errors"
	"runtime"
	"testing"
	"time"
)

type errFetcher struct{ err error }

func (e *errFetcher) Fetch(_ context.Context, _ int) ([]string, bool, error) {
	return nil, false, e.err
}

func TestGenerator(t *testing.T) {
	t.Run("fonte finita emite todas as páginas e fecha", func(t *testing.T) {
		ctx := context.Background()
		f := &StaticFetcher{Pages: [][]string{{"a"}, {"b", "c"}, {"d"}}}
		var got int
		for p := range PageGenerator(ctx, f) {
			if p.Err != nil {
				t.Fatalf("erro inesperado: %v", p.Err)
			}
			got++
		}
		if got != 3 {
			t.Fatalf("esperava 3 páginas, obteve %d", got)
		}
	})

	t.Run("Take limita o consumo em stream infinito", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		gen := PageGenerator(ctx, &TokenFetcher{Prefix: "x", Size: 2})
		pages := Take(ctx, gen, 5)
		if len(pages) != 5 {
			t.Fatalf("esperava 5 páginas, obteve %d", len(pages))
		}
		for i, p := range pages {
			if p.Number != i+1 {
				t.Fatalf("página %d errada: %+v", i, p)
			}
			if len(p.Items) != 2 {
				t.Fatalf("itens esperados=2, obteve %d", len(p.Items))
			}
		}
	})

	t.Run("cancel interrompe stream infinito sem vazar", func(t *testing.T) {
		before := runtime.NumGoroutine()
		ctx, cancel := context.WithCancel(context.Background())
		gen := PageGenerator(ctx, &TokenFetcher{Prefix: "x", Size: 1})
		// consome algumas páginas
		_ = Take(ctx, gen, 3)
		cancel()
		// drena canal
		for range gen {
		}
		time.Sleep(50 * time.Millisecond)
		after := runtime.NumGoroutine()
		if after > before+2 {
			t.Fatalf("vazamento: antes=%d depois=%d", before, after)
		}
	})

	t.Run("erro propaga e fecha canal", func(t *testing.T) {
		ctx := context.Background()
		sentinel := errors.New("boom")
		gen := PageGenerator(ctx, &errFetcher{err: sentinel})
		p, ok := <-gen
		if !ok {
			t.Fatal("esperava receber página")
		}
		if !errors.Is(p.Err, sentinel) {
			t.Fatalf("esperava sentinel, veio %v", p.Err)
		}
		// após erro, canal fecha
		if _, ok := <-gen; ok {
			t.Fatal("canal deveria ter fechado após erro")
		}
	})

	t.Run("Take respeita context cancelado", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		gen := PageGenerator(ctx, &TokenFetcher{Prefix: "y", Size: 1})
		cancel()
		pages := Take(ctx, gen, 10)
		if len(pages) > 1 {
			t.Fatalf("esperava 0 ou 1 página após cancel, obteve %d", len(pages))
		}
		// drena
		for range gen {
		}
	})

	t.Run("StaticFetcher retorna vazio fora do range", func(t *testing.T) {
		f := &StaticFetcher{Pages: [][]string{{"a"}}}
		items, more, err := f.Fetch(context.Background(), 99)
		if err != nil || more || len(items) != 0 {
			t.Fatalf("resposta inesperada: items=%v more=%v err=%v", items, more, err)
		}
	})
}
