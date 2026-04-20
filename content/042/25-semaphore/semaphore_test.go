package main

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSemaphore(t *testing.T) {
	t.Run("acquire respeita context cancelado", func(t *testing.T) {
		s := NewSemaphore(1)
		if err := s.Acquire(context.Background()); err != nil {
			t.Fatalf("acquire inesperado: %v", err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
		defer cancel()
		if err := s.Acquire(ctx); err == nil {
			t.Fatal("esperava erro de contexto")
		}
		s.Release()
	})

	t.Run("release sem acquire faz panic", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Fatal("esperava panic")
			}
		}()
		s := NewSemaphore(1)
		s.Release()
	})

	t.Run("limitedrun não ultrapassa N simultâneos", func(t *testing.T) {
		const max = 3
		const total = 20
		var inFlight, peak atomic.Int64
		calls := make([]Caller, total)
		for i := range calls {
			calls[i] = APICaller(15*time.Millisecond, &inFlight, &peak)
		}
		resps, errs := LimitedRun(context.Background(), max, calls)
		for i, e := range errs {
			if e != nil {
				t.Fatalf("erro em %d: %v", i, e)
			}
		}
		if len(resps) != total {
			t.Fatalf("esperava %d respostas, obteve %d", total, len(resps))
		}
		if p := peak.Load(); p > int64(max) {
			t.Fatalf("concorrência excedeu limite: peak=%d max=%d", p, max)
		}
	})

	t.Run("cancelamento não vaza goroutines", func(t *testing.T) {
		before := runtime.NumGoroutine()
		var inFlight, peak atomic.Int64
		calls := make([]Caller, 30)
		for i := range calls {
			calls[i] = APICaller(100*time.Millisecond, &inFlight, &peak)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		_, _ = LimitedRun(ctx, 4, calls)
		time.Sleep(80 * time.Millisecond)
		after := runtime.NumGoroutine()
		if after > before+2 {
			t.Fatalf("vazamento: antes=%d depois=%d", before, after)
		}
	})

	t.Run("concorrência real entre goroutines", func(t *testing.T) {
		s := NewSemaphore(2)
		var wg sync.WaitGroup
		var peak atomic.Int64
		var cur atomic.Int64
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := s.Acquire(context.Background()); err != nil {
					t.Errorf("acquire: %v", err)
					return
				}
				defer s.Release()
				c := cur.Add(1)
				for {
					p := peak.Load()
					if c <= p || peak.CompareAndSwap(p, c) {
						break
					}
				}
				time.Sleep(10 * time.Millisecond)
				cur.Add(-1)
			}()
		}
		wg.Wait()
		if p := peak.Load(); p > 2 {
			t.Fatalf("limite violado: peak=%d", p)
		}
	})

	t.Run("n inválido vira 1", func(t *testing.T) {
		s := NewSemaphore(0)
		if cap(s.slots) != 1 {
			t.Fatalf("esperava cap=1, obteve %d", cap(s.slots))
		}
	})
}
