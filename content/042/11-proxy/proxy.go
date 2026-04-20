package main

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// Quote é a cotação devolvida pela API.
type Quote struct {
	Symbol string
	Price  float64
	AsOf   time.Time
}

// QuoteAPI é a interface comum entre proxy e real subject.
type QuoteAPI interface {
	Get(ctx context.Context, symbol string) (Quote, error)
}

// ExternalQuoteAPI simula a API externa custosa (lenta e rate-limited no upstream).
type ExternalQuoteAPI struct {
	Calls    atomic.Int64
	Latency  time.Duration
	Prices   map[string]float64
	Now      func() time.Time
	FailNext atomic.Bool
}

func NewExternalQuoteAPI(prices map[string]float64) *ExternalQuoteAPI {
	return &ExternalQuoteAPI{
		Latency: 10 * time.Millisecond,
		Prices:  prices,
		Now:     time.Now,
	}
}

func (e *ExternalQuoteAPI) Get(ctx context.Context, symbol string) (Quote, error) {
	e.Calls.Add(1)
	if e.FailNext.Swap(false) {
		return Quote{}, errors.New("upstream error")
	}
	select {
	case <-ctx.Done():
		return Quote{}, ctx.Err()
	case <-time.After(e.Latency):
	}
	price, ok := e.Prices[symbol]
	if !ok {
		return Quote{}, errors.New("symbol not found")
	}
	return Quote{Symbol: symbol, Price: price, AsOf: e.Now()}, nil
}

// cacheEntry guarda o valor e expiração.
type cacheEntry struct {
	quote   Quote
	expires time.Time
}

// CachingRateLimitedProxy combina cache em memória com rate-limit na frente
// do real subject (ExternalQuoteAPI).
type CachingRateLimitedProxy struct {
	real QuoteAPI
	ttl  time.Duration
	now  func() time.Time

	mu    sync.Mutex
	cache map[string]cacheEntry

	// token bucket simples
	limitMu    sync.Mutex
	tokens     float64
	max        float64
	refillRate float64
	lastRefill time.Time
}

func NewCachingRateLimitedProxy(real QuoteAPI, ttl time.Duration, maxTokens, refillPerSecond float64) *CachingRateLimitedProxy {
	now := time.Now
	return &CachingRateLimitedProxy{
		real:       real,
		ttl:        ttl,
		now:        now,
		cache:      map[string]cacheEntry{},
		tokens:     maxTokens,
		max:        maxTokens,
		refillRate: refillPerSecond,
		lastRefill: now(),
	}
}

// SetClock permite controlar o tempo em testes.
func (p *CachingRateLimitedProxy) SetClock(f func() time.Time) {
	p.now = f
	p.limitMu.Lock()
	p.lastRefill = f()
	p.limitMu.Unlock()
}

func (p *CachingRateLimitedProxy) Get(ctx context.Context, symbol string) (Quote, error) {
	if symbol == "" {
		return Quote{}, errors.New("symbol required")
	}

	now := p.now()

	// 1. cache hit?
	p.mu.Lock()
	if e, ok := p.cache[symbol]; ok && now.Before(e.expires) {
		p.mu.Unlock()
		return e.quote, nil
	}
	p.mu.Unlock()

	// 2. rate-limit antes de sair pra rede
	if !p.allow(now) {
		return Quote{}, ErrRateLimited
	}

	// 3. chama o real subject
	q, err := p.real.Get(ctx, symbol)
	if err != nil {
		return Quote{}, err
	}

	// 4. grava no cache
	p.mu.Lock()
	p.cache[symbol] = cacheEntry{quote: q, expires: now.Add(p.ttl)}
	p.mu.Unlock()

	return q, nil
}

// ErrRateLimited é retornado quando o proxy barra a chamada.
var ErrRateLimited = errors.New("proxy: rate limited")

func (p *CachingRateLimitedProxy) allow(now time.Time) bool {
	p.limitMu.Lock()
	defer p.limitMu.Unlock()
	elapsed := now.Sub(p.lastRefill).Seconds()
	if elapsed > 0 {
		p.tokens += elapsed * p.refillRate
		if p.tokens > p.max {
			p.tokens = p.max
		}
		p.lastRefill = now
	}
	if p.tokens >= 1 {
		p.tokens--
		return true
	}
	return false
}

// Invalidate remove uma chave do cache.
func (p *CachingRateLimitedProxy) Invalidate(symbol string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.cache, symbol)
}
