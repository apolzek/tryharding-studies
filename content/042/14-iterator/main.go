package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
)

// pagesDataset serve dados paginados simulados.
func pagesDataset() map[string]pageResponse {
	return map[string]pageResponse{
		"": {Items: []Item{{ID: "1", Name: "a"}, {ID: "2", Name: "b"}}, NextCursor: "p2"},
		"p2": {Items: []Item{{ID: "3", Name: "c"}, {ID: "4", Name: "d"}}, NextCursor: "p3"},
		"p3": {Items: []Item{{ID: "5", Name: "e"}}, NextCursor: ""},
	}
}

func fakeServer(data map[string]pageResponse) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := r.URL.Query().Get("cursor")
		page, ok := data[c]
		if !ok {
			http.Error(w, "cursor invalido", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(page)
	}))
}

func main() {
	srv := fakeServer(pagesDataset())
	defer srv.Close()

	ctx := context.Background()
	it := NewPageIterator(srv.Client(), srv.URL+"/items")
	for {
		item, ok := it.Next(ctx)
		if !ok {
			break
		}
		fmt.Printf("item id=%s name=%s\n", item.ID, item.Name)
	}
	if err := it.Err(); err != nil {
		fmt.Println("erro:", err)
	}
}
