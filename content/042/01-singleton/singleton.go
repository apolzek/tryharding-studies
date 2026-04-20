package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// DBConfig representa configuração necessária para abrir um pool.
type DBConfig struct {
	DSN         string
	MaxOpen     int
	DialTimeout time.Duration
}

// Conn simula uma conexão retirada do pool.
type Conn struct {
	id   int
	pool *DBPool
}

func (c *Conn) Ping(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func (c *Conn) Release() {
	c.pool.release(c)
}

// DBPool é o recurso caro que queremos inicializar uma única vez.
type DBPool struct {
	cfg      DBConfig
	mu       sync.Mutex
	inUse    map[int]*Conn
	next     int64
	maxOpen  int
	acquired int64
}

func (p *DBPool) Acquire(ctx context.Context) (*Conn, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.inUse) >= p.maxOpen {
		return nil, errors.New("pool esgotado")
	}
	id := atomic.AddInt64(&p.next, 1)
	c := &Conn{id: int(id), pool: p}
	p.inUse[int(id)] = c
	atomic.AddInt64(&p.acquired, 1)
	return c, nil
}

func (p *DBPool) release(c *Conn) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.inUse, c.id)
}

func (p *DBPool) Stats() (inUse int, totalAcquired int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.inUse), atomic.LoadInt64(&p.acquired)
}

// Estado do singleton.
var (
	instance *DBPool
	initOnce sync.Once
	initErr  error
)

// GetPool devolve o pool global. A inicialização acontece uma única vez.
func GetPool(cfg DBConfig) (*DBPool, error) {
	initOnce.Do(func() {
		if cfg.DSN == "" {
			initErr = errors.New("DSN vazio")
			return
		}
		if cfg.MaxOpen <= 0 {
			cfg.MaxOpen = 10
		}
		instance = &DBPool{
			cfg:     cfg,
			inUse:   make(map[int]*Conn),
			maxOpen: cfg.MaxOpen,
		}
	})
	if initErr != nil {
		return nil, fmt.Errorf("inicializando pool: %w", initErr)
	}
	return instance, nil
}

// resetForTests é usado apenas pelos testes para reciclar o singleton.
func resetForTests() {
	instance = nil
	initOnce = sync.Once{}
	initErr = nil
}
