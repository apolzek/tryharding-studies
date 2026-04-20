package main

import (
	"context"
	"fmt"
	"time"
)

func main() {
	// Exemplo 1: paginação finita.
	ctx1, cancel1 := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel1()

	finite := &StaticFetcher{Pages: [][]string{
		{"user-1", "user-2"},
		{"user-3", "user-4"},
		{"user-5"},
	}}
	gen := PageGenerator(ctx1, finite)
	for p := range gen {
		if p.Err != nil {
			fmt.Printf("erro na página %d: %v\n", p.Number, p.Err)
			continue
		}
		fmt.Printf("pagina=%d itens=%v\n", p.Number, p.Items)
	}
	fmt.Println("---")

	// Exemplo 2: stream infinito com Take para controlar consumo.
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	tokens := &TokenFetcher{Prefix: "tok", Size: 2}
	stream := PageGenerator(ctx2, tokens)
	for _, p := range Take(ctx2, stream, 3) {
		fmt.Printf("stream pagina=%d itens=%v\n", p.Number, p.Items)
	}
	cancel2() // interrompe o gerador infinito
	fmt.Println("gerador encerrado por cancel")
}
