package main

import (
	"context"
	"errors"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
)

func TestWorkerPool(t *testing.T) {
	t.Run("processa todos os jobs", func(t *testing.T) {
		ctx := context.Background()
		var count int64
		proc := func(_ context.Context, j Job) (string, error) {
			atomic.AddInt64(&count, 1)
			return j.Payload, nil
		}
		p := NewPool(3, 4, proc)
		p.Start(ctx)

		go func() {
			defer p.Stop()
			for i := 0; i < 50; i++ {
				_ = p.Submit(ctx, Job{ID: i, Payload: "x"})
			}
		}()

		var got int
		for range p.Results() {
			got++
		}
		if got != 50 {
			t.Fatalf("esperava 50 resultados, obteve %d", got)
		}
		if atomic.LoadInt64(&count) != 50 {
			t.Fatalf("esperava 50 execuções, obteve %d", count)
		}
	})

	t.Run("propaga erros do processor", func(t *testing.T) {
		ctx := context.Background()
		sentinel := errors.New("falhou")
		proc := func(_ context.Context, _ Job) (string, error) { return "", sentinel }
		p := NewPool(2, 2, proc)
		p.Start(ctx)

		go func() {
			defer p.Stop()
			_ = p.Submit(ctx, Job{ID: 1})
		}()

		r := <-p.Results()
		if !errors.Is(r.Err, sentinel) {
			t.Fatalf("esperava sentinel, veio %v", r.Err)
		}
		// drena
		for range p.Results() {
		}
	})

	t.Run("shutdown via context não vaza goroutines", func(t *testing.T) {
		before := runtime.NumGoroutine()
		ctx, cancel := context.WithCancel(context.Background())

		proc := func(ctx context.Context, _ Job) (string, error) {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(200 * time.Millisecond):
				return "ok", nil
			}
		}
		p := NewPool(4, 0, proc)
		p.Start(ctx)

		go func() {
			for i := 0; i < 20; i++ {
				if err := p.Submit(ctx, Job{ID: i}); err != nil {
					return
				}
			}
		}()

		time.Sleep(30 * time.Millisecond)
		cancel()

		// Aguarda encerramento das goroutines dos workers.
		done := make(chan struct{})
		go func() {
			p.wg.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("workers não encerraram no cancel")
		}

		// Espera um pouco para o runtime limpar.
		time.Sleep(50 * time.Millisecond)
		after := runtime.NumGoroutine()
		if after > before+2 {
			t.Fatalf("possível vazamento: antes=%d depois=%d", before, after)
		}
	})

	t.Run("concorrência real com múltiplos workers", func(t *testing.T) {
		ctx := context.Background()
		var inFlight, maxInFlight int64
		proc := func(_ context.Context, _ Job) (string, error) {
			cur := atomic.AddInt64(&inFlight, 1)
			for {
				m := atomic.LoadInt64(&maxInFlight)
				if cur <= m || atomic.CompareAndSwapInt64(&maxInFlight, m, cur) {
					break
				}
			}
			time.Sleep(20 * time.Millisecond)
			atomic.AddInt64(&inFlight, -1)
			return "", nil
		}
		p := NewPool(5, 10, proc)
		p.Start(ctx)
		go func() {
			defer p.Stop()
			for i := 0; i < 20; i++ {
				_ = p.Submit(ctx, Job{ID: i})
			}
		}()
		for range p.Results() {
		}
		if atomic.LoadInt64(&maxInFlight) < 2 {
			t.Fatalf("esperava concorrência ≥ 2, obteve %d", maxInFlight)
		}
	})
}
