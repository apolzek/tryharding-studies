package main

import (
	"context"
	"fmt"
	"sync"
)

// Job representa uma unidade de trabalho (ex.: um log a processar).
type Job struct {
	ID      int
	Payload string
}

// Result representa a saída produzida por um worker.
type Result struct {
	JobID  int
	Output string
	Err    error
}

// Processor é a função de negócio executada por cada worker.
type Processor func(ctx context.Context, j Job) (string, error)

// Pool encapsula workers, canais de entrada/saída e ciclo de vida.
type Pool struct {
	workers int
	jobs    chan Job
	results chan Result
	wg      sync.WaitGroup
	proc    Processor
}

// NewPool cria um pool com N workers e canais dimensionados.
func NewPool(workers int, buffer int, proc Processor) *Pool {
	if workers <= 0 {
		workers = 1
	}
	if buffer < 0 {
		buffer = 0
	}
	return &Pool{
		workers: workers,
		jobs:    make(chan Job, buffer),
		results: make(chan Result, buffer),
		proc:    proc,
	}
}

// Start inicia as goroutines dos workers.
func (p *Pool) Start(ctx context.Context) {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(ctx, i)
	}
}

func (p *Pool) worker(ctx context.Context, id int) {
	defer p.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-p.jobs:
			if !ok {
				return
			}
			out, err := p.proc(ctx, job)
			select {
			case <-ctx.Done():
				return
			case p.results <- Result{JobID: job.ID, Output: out, Err: err}:
			}
		}
	}
}

// Submit envia um job respeitando cancelamento.
func (p *Pool) Submit(ctx context.Context, j Job) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case p.jobs <- j:
		return nil
	}
}

// Results expõe o canal de saída (somente leitura).
func (p *Pool) Results() <-chan Result {
	return p.results
}

// Stop sinaliza fim dos jobs e aguarda drenagem.
func (p *Pool) Stop() {
	close(p.jobs)
	p.wg.Wait()
	close(p.results)
}

// DefaultProcessor é um exemplo simples usado na demonstração.
func DefaultProcessor(_ context.Context, j Job) (string, error) {
	return fmt.Sprintf("job=%d processed payload=%q", j.ID, j.Payload), nil
}
