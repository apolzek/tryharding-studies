package main

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Semaphore limita o número de operações concorrentes via channel buferizado.
type Semaphore struct {
	slots chan struct{}
}

// NewSemaphore cria um semáforo com N permits.
func NewSemaphore(n int) *Semaphore {
	if n < 1 {
		n = 1
	}
	return &Semaphore{slots: make(chan struct{}, n)}
}

// Acquire adquire um permit, respeitando cancelamento.
func (s *Semaphore) Acquire(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case s.slots <- struct{}{}:
		return nil
	}
}

// Release libera um permit. Chamar sem Acquire prévio faz panic.
func (s *Semaphore) Release() {
	select {
	case <-s.slots:
	default:
		panic("semaphore: release sem acquire")
	}
}

// InFlight devolve quantos permits estão em uso no momento.
func (s *Semaphore) InFlight() int { return len(s.slots) }

// Caller representa um cliente HTTP externo (abstraído).
type Caller func(ctx context.Context, id int) (string, error)

// LimitedRun executa `calls` em paralelo, mas com no máximo `max` simultâneos.
// Retorna slice de respostas na ordem de conclusão e slice de erros casados.
func LimitedRun(ctx context.Context, max int, calls []Caller) ([]string, []error) {
	sem := NewSemaphore(max)
	var wg sync.WaitGroup
	responses := make([]string, len(calls))
	errs := make([]error, len(calls))

	for i, call := range calls {
		if err := sem.Acquire(ctx); err != nil {
			errs[i] = err
			continue
		}
		wg.Add(1)
		go func(i int, call Caller) {
			defer wg.Done()
			defer sem.Release()
			resp, err := call(ctx, i)
			responses[i] = resp
			errs[i] = err
		}(i, call)
	}
	wg.Wait()
	return responses, errs
}

// APICaller devolve um Caller que simula uma chamada externa com latência.
func APICaller(d time.Duration, counter *atomic.Int64, peak *atomic.Int64) Caller {
	return func(ctx context.Context, id int) (string, error) {
		cur := counter.Add(1)
		for {
			p := peak.Load()
			if cur <= p || peak.CompareAndSwap(p, cur) {
				break
			}
		}
		defer counter.Add(-1)

		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(d):
			return fmt.Sprintf("resp-%d", id), nil
		}
	}
}
