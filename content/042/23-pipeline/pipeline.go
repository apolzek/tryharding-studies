package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// Event representa um registro cru do log ingerido.
type Event struct {
	Raw string
}

// Parsed é o evento após parsing estruturado.
type Parsed struct {
	User   string
	Action string
	Amount int
}

// Enriched é o evento após lookup/merge de metadados.
type Enriched struct {
	Parsed
	Tier string
}

// Persisted é o resultado final do estágio de saída.
type Persisted struct {
	Enriched
	ID int
}

// Ingest emite eventos crus respeitando cancelamento.
func Ingest(ctx context.Context, raw []string) <-chan Event {
	out := make(chan Event)
	go func() {
		defer close(out)
		for _, r := range raw {
			select {
			case <-ctx.Done():
				return
			case out <- Event{Raw: r}:
			}
		}
	}()
	return out
}

// Parse converte linhas cru em Parsed. Linhas inválidas são descartadas.
func Parse(ctx context.Context, in <-chan Event) <-chan Parsed {
	out := make(chan Parsed)
	go func() {
		defer close(out)
		for ev := range in {
			parts := strings.Split(ev.Raw, ",")
			if len(parts) != 3 {
				continue
			}
			amount, err := strconv.Atoi(strings.TrimSpace(parts[2]))
			if err != nil {
				continue
			}
			p := Parsed{
				User:   strings.TrimSpace(parts[0]),
				Action: strings.TrimSpace(parts[1]),
				Amount: amount,
			}
			select {
			case <-ctx.Done():
				return
			case out <- p:
			}
		}
	}()
	return out
}

// Enrich adiciona o tier do usuário (lookup simulado).
func Enrich(ctx context.Context, in <-chan Parsed) <-chan Enriched {
	out := make(chan Enriched)
	tier := func(amount int) string {
		switch {
		case amount >= 1000:
			return "gold"
		case amount >= 100:
			return "silver"
		default:
			return "bronze"
		}
	}
	go func() {
		defer close(out)
		for p := range in {
			e := Enriched{Parsed: p, Tier: tier(p.Amount)}
			select {
			case <-ctx.Done():
				return
			case out <- e:
			}
		}
	}()
	return out
}

// Persist simula persistência atribuindo IDs sequenciais.
func Persist(ctx context.Context, in <-chan Enriched) <-chan Persisted {
	out := make(chan Persisted)
	go func() {
		defer close(out)
		id := 0
		for e := range in {
			id++
			rec := Persisted{Enriched: e, ID: id}
			select {
			case <-ctx.Done():
				return
			case out <- rec:
			}
		}
	}()
	return out
}

// Format gera uma linha legível para stdout.
func Format(p Persisted) string {
	return fmt.Sprintf("id=%d user=%s action=%s amount=%d tier=%s",
		p.ID, p.User, p.Action, p.Amount, p.Tier)
}
