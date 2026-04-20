package main

import (
	"context"
	"fmt"
)

// Page representa uma página de resultados paginados.
type Page struct {
	Number int
	Items  []string
	Err    error
}

// Fetcher abstrai a fonte paginada (API, banco, SDK).
type Fetcher interface {
	Fetch(ctx context.Context, page int) ([]string, bool, error)
}

// PageGenerator devolve um canal que emite páginas preguiçosamente.
// Cada leitura no canal dispara o próximo Fetch. Fechar via `ctx` para o fluxo.
func PageGenerator(ctx context.Context, f Fetcher) <-chan Page {
	out := make(chan Page)
	go func() {
		defer close(out)
		page := 1
		for {
			items, hasMore, err := f.Fetch(ctx, page)
			p := Page{Number: page, Items: items, Err: err}
			select {
			case <-ctx.Done():
				return
			case out <- p:
			}
			if err != nil || !hasMore {
				return
			}
			page++
		}
	}()
	return out
}

// Take consome no máximo n páginas do gerador (helper para controle de fluxo).
func Take(ctx context.Context, g <-chan Page, n int) []Page {
	out := make([]Page, 0, n)
	for i := 0; i < n; i++ {
		select {
		case <-ctx.Done():
			return out
		case p, ok := <-g:
			if !ok {
				return out
			}
			out = append(out, p)
		}
	}
	return out
}

// StaticFetcher é uma implementação em memória para demos e testes.
type StaticFetcher struct {
	Pages [][]string
}

func (s *StaticFetcher) Fetch(ctx context.Context, page int) ([]string, bool, error) {
	select {
	case <-ctx.Done():
		return nil, false, ctx.Err()
	default:
	}
	idx := page - 1
	if idx < 0 || idx >= len(s.Pages) {
		return nil, false, nil
	}
	hasMore := idx < len(s.Pages)-1
	return s.Pages[idx], hasMore, nil
}

// TokenFetcher gera tokens sintéticos indefinidamente (stream infinito).
type TokenFetcher struct {
	Prefix string
	Size   int
}

func (t *TokenFetcher) Fetch(ctx context.Context, page int) ([]string, bool, error) {
	select {
	case <-ctx.Done():
		return nil, false, ctx.Err()
	default:
	}
	size := t.Size
	if size <= 0 {
		size = 3
	}
	items := make([]string, size)
	for i := 0; i < size; i++ {
		items[i] = fmt.Sprintf("%s-p%d-t%d", t.Prefix, page, i)
	}
	return items, true, nil // stream infinito
}
