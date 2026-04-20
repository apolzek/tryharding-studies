package main

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func newTestAPI() *ExternalQuoteAPI {
	api := NewExternalQuoteAPI(map[string]float64{"X": 10, "Y": 20})
	api.Latency = time.Microsecond
	fixed := time.Unix(1_700_000_000, 0)
	api.Now = func() time.Time { return fixed }
	return api
}

func TestProxy_Get(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		symbol  string
		wantErr bool
		want    float64
	}{
		{"hit existing symbol", "X", false, 10},
		{"hit other symbol", "Y", false, 20},
		{"unknown symbol", "ZZZ", true, 0},
		{"empty symbol rejected", "", true, 0},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			real := newTestAPI()
			p := NewCachingRateLimitedProxy(real, time.Minute, 10, 10)
			q, err := p.Get(context.Background(), tc.symbol)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected: %v", err)
			}
			if q.Price != tc.want {
				t.Fatalf("price=%.2f want %.2f", q.Price, tc.want)
			}
		})
	}
}

func TestProxy_CachesResult(t *testing.T) {
	t.Parallel()
	real := newTestAPI()
	p := NewCachingRateLimitedProxy(real, time.Minute, 10, 10)

	for i := 0; i < 5; i++ {
		if _, err := p.Get(context.Background(), "X"); err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}
	if got := real.Calls.Load(); got != 1 {
		t.Fatalf("upstream calls=%d want 1", got)
	}
}

func TestProxy_CacheExpires(t *testing.T) {
	t.Parallel()
	real := newTestAPI()
	p := NewCachingRateLimitedProxy(real, 100*time.Millisecond, 10, 10)

	now := time.Unix(100, 0)
	p.SetClock(func() time.Time { return now })

	if _, err := p.Get(context.Background(), "X"); err != nil {
		t.Fatal(err)
	}
	// Avança o relógio além do TTL.
	now = now.Add(200 * time.Millisecond)
	if _, err := p.Get(context.Background(), "X"); err != nil {
		t.Fatal(err)
	}
	if got := real.Calls.Load(); got != 2 {
		t.Fatalf("upstream calls=%d want 2", got)
	}
}

func TestProxy_RateLimited(t *testing.T) {
	t.Parallel()
	real := newTestAPI()
	// Apenas 2 tokens, refill quase zero para simular bucket estourado.
	p := NewCachingRateLimitedProxy(real, 0, 2, 0.0001)
	// TTL zero força todas as chamadas a caírem no upstream.

	for i := 0; i < 2; i++ {
		sym := "X"
		if i == 1 {
			sym = "Y"
		}
		if _, err := p.Get(context.Background(), sym); err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}
	_, err := p.Get(context.Background(), "X")
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited, got %v", err)
	}
}

func TestProxy_Invalidate(t *testing.T) {
	t.Parallel()
	real := newTestAPI()
	p := NewCachingRateLimitedProxy(real, time.Minute, 10, 10)

	_, _ = p.Get(context.Background(), "X")
	p.Invalidate("X")
	_, _ = p.Get(context.Background(), "X")
	if got := real.Calls.Load(); got != 2 {
		t.Fatalf("upstream calls=%d want 2", got)
	}
}

func TestProxy_UpstreamError(t *testing.T) {
	t.Parallel()
	real := newTestAPI()
	real.FailNext.Store(true)
	p := NewCachingRateLimitedProxy(real, time.Minute, 10, 10)

	_, err := p.Get(context.Background(), "X")
	if err == nil {
		t.Fatal("expected upstream error")
	}
	// A falha não deve ser cacheada.
	if _, err := p.Get(context.Background(), "X"); err != nil {
		t.Fatalf("second call should succeed: %v", err)
	}
}

func TestProxy_Concurrent(t *testing.T) {
	t.Parallel()
	real := newTestAPI()
	p := NewCachingRateLimitedProxy(real, time.Minute, 100, 100)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = p.Get(context.Background(), "X")
		}()
	}
	wg.Wait()
	// Não podemos afirmar "calls == 1" porque a corrida pode deixar algumas
	// passarem antes do cache ser populado, mas devem ser muito menos que 50.
	if got := real.Calls.Load(); got >= 50 {
		t.Fatalf("expected caching to reduce upstream calls, got %d", got)
	}
}
