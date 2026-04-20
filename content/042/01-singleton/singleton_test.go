package main

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestGetPool(t *testing.T) {
	tests := []struct {
		name    string
		cfg     DBConfig
		wantErr bool
	}{
		{name: "ok", cfg: DBConfig{DSN: "x", MaxOpen: 2}, wantErr: false},
		{name: "dsn vazio", cfg: DBConfig{DSN: "", MaxOpen: 2}, wantErr: true},
		{name: "maxopen padrao", cfg: DBConfig{DSN: "y"}, wantErr: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resetForTests()
			_, err := GetPool(tc.cfg)
			if (err != nil) != tc.wantErr {
				t.Fatalf("esperado erro=%v, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestSingletonSameInstance(t *testing.T) {
	resetForTests()
	p1, err := GetPool(DBConfig{DSN: "z", MaxOpen: 3})
	if err != nil {
		t.Fatal(err)
	}
	p2, err := GetPool(DBConfig{DSN: "ignorado"})
	if err != nil {
		t.Fatal(err)
	}
	if p1 != p2 {
		t.Fatal("ponteiros diferentes: singleton falhou")
	}
}

func TestPoolConcurrency(t *testing.T) {
	resetForTests()
	p, err := GetPool(DBConfig{DSN: "c", MaxOpen: 50})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			c, err := p.Acquire(ctx)
			if err != nil {
				return
			}
			_ = c.Ping(ctx)
			c.Release()
		}()
	}
	wg.Wait()
	inUse, _ := p.Stats()
	if inUse != 0 {
		t.Fatalf("esperado 0 em uso, got %d", inUse)
	}
}

func TestPoolExhaustion(t *testing.T) {
	resetForTests()
	p, err := GetPool(DBConfig{DSN: "d", MaxOpen: 1})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	c1, err := p.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Acquire(ctx); err == nil {
		t.Fatal("esperado erro de pool esgotado")
	}
	c1.Release()
	if _, err := p.Acquire(ctx); err != nil {
		t.Fatalf("após release deveria adquirir, got %v", err)
	}
}

func TestPoolCtxCanceled(t *testing.T) {
	resetForTests()
	p, err := GetPool(DBConfig{DSN: "e", MaxOpen: 1})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := p.Acquire(ctx); err == nil {
		t.Fatal("esperado erro de contexto cancelado")
	}
}

func TestOnceInitOnlyOnce(t *testing.T) {
	resetForTests()
	var wg sync.WaitGroup
	pointers := make([]*DBPool, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			p, _ := GetPool(DBConfig{DSN: "f", MaxOpen: 5})
			pointers[i] = p
		}(i)
	}
	wg.Wait()
	for i := 1; i < len(pointers); i++ {
		if pointers[i] != pointers[0] {
			t.Fatalf("ponteiro %d diferente do 0", i)
		}
	}
}
