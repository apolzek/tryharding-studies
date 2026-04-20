package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// Item representa um elemento retornado pela API.
type Item struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// pageResponse representa o payload paginado com cursor.
type pageResponse struct {
	Items      []Item `json:"items"`
	NextCursor string `json:"next_cursor"`
}

// PageIterator abstrai a paginacao cursor-based.
type PageIterator struct {
	client  *http.Client
	baseURL string
	cursor  string
	buffer  []Item
	done    bool
	err     error
}

// NewPageIterator cria um iterator para o endpoint informado.
func NewPageIterator(client *http.Client, baseURL string) *PageIterator {
	if client == nil {
		client = http.DefaultClient
	}
	return &PageIterator{client: client, baseURL: baseURL}
}

// Next retorna o proximo item, buscando nova pagina sob demanda.
// Retorna (item, true) quando ha item; (zero, false) no fim ou em erro.
func (it *PageIterator) Next(ctx context.Context) (Item, bool) {
	if it.err != nil {
		return Item{}, false
	}
	if len(it.buffer) == 0 {
		if it.done {
			return Item{}, false
		}
		if err := it.fetch(ctx); err != nil {
			it.err = err
			return Item{}, false
		}
		if len(it.buffer) == 0 {
			return Item{}, false
		}
	}
	item := it.buffer[0]
	it.buffer = it.buffer[1:]
	return item, true
}

// Err retorna o erro acumulado.
func (it *PageIterator) Err() error { return it.err }

func (it *PageIterator) fetch(ctx context.Context) error {
	u, err := url.Parse(it.baseURL)
	if err != nil {
		return err
	}
	q := u.Query()
	if it.cursor != "" {
		q.Set("cursor", it.cursor)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	resp, err := it.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}

	var page pageResponse
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return errors.Join(errors.New("decode falhou"), err)
	}
	it.buffer = append(it.buffer, page.Items...)
	it.cursor = page.NextCursor
	if page.NextCursor == "" {
		it.done = true
	}
	return nil
}

// Collect consome todo o iterator em slice (uso cuidadoso com paginas grandes).
func Collect(ctx context.Context, it *PageIterator) ([]Item, error) {
	var out []Item
	for {
		item, ok := it.Next(ctx)
		if !ok {
			break
		}
		out = append(out, item)
	}
	return out, it.Err()
}
