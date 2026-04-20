package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// Quote representa uma cotação obtida de um provedor.
type Quote struct {
	Provider string
	Symbol   string
	Price    float64
	Err      error
}

// Provider é a interface de cada fonte de cotação.
type Provider interface {
	Name() string
	Fetch(ctx context.Context, symbol string) (float64, error)
}

// FakeProvider é uma implementação determinística para demos e testes.
type FakeProvider struct {
	ID    string
	Delta float64
	Fail  bool
}

func (p FakeProvider) Name() string { return p.ID }

func (p FakeProvider) Fetch(ctx context.Context, symbol string) (float64, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}
	if p.Fail {
		return 0, fmt.Errorf("%s indisponível", p.ID)
	}
	base := 100.0
	if symbol == "BTC" {
		base = 50000
	}
	return base + p.Delta, nil
}

// fanOut dispara uma goroutine por provedor, retornando um canal por provedor.
func fanOut(ctx context.Context, providers []Provider, symbol string) []<-chan Quote {
	out := make([]<-chan Quote, 0, len(providers))
	for _, p := range providers {
		ch := make(chan Quote, 1)
		out = append(out, ch)
		go func(p Provider, ch chan<- Quote) {
			defer close(ch)
			price, err := p.Fetch(ctx, symbol)
			q := Quote{Provider: p.Name(), Symbol: symbol, Price: price, Err: err}
			select {
			case <-ctx.Done():
			case ch <- q:
			}
		}(p, ch)
	}
	return out
}

// fanIn multiplexa vários canais em um só, fechando quando todos fecharem.
func fanIn(ctx context.Context, chans []<-chan Quote) <-chan Quote {
	out := make(chan Quote)
	var wg sync.WaitGroup
	wg.Add(len(chans))
	for _, c := range chans {
		go func(c <-chan Quote) {
			defer wg.Done()
			for q := range c {
				select {
				case <-ctx.Done():
					return
				case out <- q:
				}
			}
		}(c)
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

// Aggregate consulta todos os provedores em paralelo e retorna quotes colhidas.
// Erros individuais não abortam o agregado.
func Aggregate(ctx context.Context, providers []Provider, symbol string) []Quote {
	quotes := fanIn(ctx, fanOut(ctx, providers, symbol))
	var result []Quote
	for q := range quotes {
		result = append(result, q)
	}
	return result
}

// BestPrice retorna a menor cotação válida, ou erro se nenhuma válida.
func BestPrice(quotes []Quote) (Quote, error) {
	var best Quote
	found := false
	for _, q := range quotes {
		if q.Err != nil {
			continue
		}
		if !found || q.Price < best.Price {
			best = q
			found = true
		}
	}
	if !found {
		return Quote{}, errors.New("nenhuma cotação válida")
	}
	return best, nil
}
