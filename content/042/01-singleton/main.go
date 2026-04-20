package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

func main() {
	cfg := DBConfig{DSN: "postgres://user:pass@localhost:5432/app", MaxOpen: 5, DialTimeout: 2 * time.Second}

	pool, err := GetPool(cfg)
	if err != nil {
		fmt.Println("erro:", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c, err := pool.Acquire(ctx)
			if err != nil {
				fmt.Printf("worker %d falhou: %v\n", i, err)
				return
			}
			defer c.Release()
			if err := c.Ping(ctx); err != nil {
				fmt.Printf("worker %d ping falhou: %v\n", i, err)
				return
			}
			fmt.Printf("worker %d usou conn #%d\n", i, c.id)
		}(i)
	}
	wg.Wait()

	// Chamada posterior devolve a mesma instância sem reinicializar.
	same, _ := GetPool(DBConfig{DSN: "ignorado"})
	fmt.Printf("mesma instância? %v\n", pool == same)

	inUse, total := pool.Stats()
	fmt.Printf("em uso: %d, total adquiridas: %d\n", inUse, total)
}
