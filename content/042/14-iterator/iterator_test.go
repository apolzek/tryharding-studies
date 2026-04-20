package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testServer(t *testing.T, data map[string]pageResponse) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := r.URL.Query().Get("cursor")
		page, ok := data[c]
		if !ok {
			http.Error(w, "bad cursor", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(page)
	}))
}

func TestIterator(t *testing.T) {
	tests := []struct {
		name string
		data map[string]pageResponse
		want int
	}{
		{
			name: "multi-page",
			data: map[string]pageResponse{
				"":   {Items: []Item{{ID: "1"}, {ID: "2"}}, NextCursor: "n1"},
				"n1": {Items: []Item{{ID: "3"}}, NextCursor: "n2"},
				"n2": {Items: []Item{{ID: "4"}, {ID: "5"}}, NextCursor: ""},
			},
			want: 5,
		},
		{
			name: "single page",
			data: map[string]pageResponse{
				"": {Items: []Item{{ID: "1"}}, NextCursor: ""},
			},
			want: 1,
		},
		{
			name: "empty",
			data: map[string]pageResponse{
				"": {Items: nil, NextCursor: ""},
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := testServer(t, tt.data)
			defer srv.Close()

			it := NewPageIterator(srv.Client(), srv.URL+"/x")
			items, err := Collect(context.Background(), it)
			if err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}
			if len(items) != tt.want {
				t.Fatalf("esperava %d itens, got %d", tt.want, len(items))
			}
		})
	}
}

func TestIteratorErro(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	it := NewPageIterator(srv.Client(), srv.URL+"/x")
	_, ok := it.Next(context.Background())
	if ok {
		t.Fatalf("nao deveria retornar item")
	}
	if it.Err() == nil {
		t.Fatalf("esperava erro preservado")
	}
}

func TestIteratorCtxCancel(t *testing.T) {
	srv := testServer(t, map[string]pageResponse{"": {Items: []Item{{ID: "1"}}}})
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	it := NewPageIterator(srv.Client(), srv.URL+"/x")
	_, ok := it.Next(ctx)
	if ok {
		t.Fatalf("nao deveria obter item com ctx cancelado")
	}
	if it.Err() == nil {
		t.Fatalf("esperava erro")
	}
}

func TestIteratorClientDefault(t *testing.T) {
	srv := testServer(t, map[string]pageResponse{"": {Items: []Item{{ID: "1"}}, NextCursor: ""}})
	defer srv.Close()
	it := NewPageIterator(nil, srv.URL+"/x")
	items, err := Collect(context.Background(), it)
	if err != nil {
		t.Fatalf("erro: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("esperava 1 item, got %d", len(items))
	}
}

func TestIteratorJSONInvalido(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("nao-json"))
	}))
	defer srv.Close()
	it := NewPageIterator(srv.Client(), srv.URL+"/x")
	_, ok := it.Next(context.Background())
	if ok {
		t.Fatalf("nao deveria obter item com json invalido")
	}
	if it.Err() == nil {
		t.Fatalf("esperava erro de decode")
	}
}

func TestIteratorURLInvalida(t *testing.T) {
	it := NewPageIterator(http.DefaultClient, "://bad-url")
	_, ok := it.Next(context.Background())
	if ok {
		t.Fatalf("nao deveria retornar item com url invalida")
	}
	if it.Err() == nil {
		t.Fatalf("esperava erro")
	}
}

func TestMainDemo(t *testing.T) {
	main()
}
