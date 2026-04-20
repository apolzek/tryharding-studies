package main

import (
	"context"
	"fmt"
	"time"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	pool := NewPool(4, 8, DefaultProcessor)
	pool.Start(ctx)

	// Produz jobs em goroutine separada.
	go func() {
		defer pool.Stop()
		for i := 1; i <= 10; i++ {
			_ = pool.Submit(ctx, Job{ID: i, Payload: fmt.Sprintf("line-%d", i)})
		}
	}()

	// Consome resultados até canal fechar.
	for r := range pool.Results() {
		if r.Err != nil {
			fmt.Printf("erro job=%d: %v\n", r.JobID, r.Err)
			continue
		}
		fmt.Println(r.Output)
	}
	fmt.Println("pool encerrado")
}
