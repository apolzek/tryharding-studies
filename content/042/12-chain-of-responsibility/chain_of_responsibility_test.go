package main

import (
	"context"
	"errors"
	"testing"
)

func TestPipeline(t *testing.T) {
	tokens := map[string]string{"tok": "alice"}

	tests := []struct {
		name    string
		req     *Request
		wantErr error
	}{
		{
			name:    "valido",
			req:     &Request{Token: "tok", ClientIP: "1.1.1.1", Body: map[string]any{"amount": 10.0, "currency": "BRL"}},
			wantErr: nil,
		},
		{
			name:    "sem token",
			req:     &Request{Token: "", ClientIP: "1.1.1.2", Body: map[string]any{"amount": 10.0, "currency": "BRL"}},
			wantErr: ErrUnauthorized,
		},
		{
			name:    "schema invalido",
			req:     &Request{Token: "tok", ClientIP: "1.1.1.3", Body: map[string]any{"amount": 10.0}},
			wantErr: ErrInvalidSchema,
		},
		{
			name:    "regra de negocio",
			req:     &Request{Token: "tok", ClientIP: "1.1.1.4", Body: map[string]any{"amount": 999999.0, "currency": "BRL"}},
			wantErr: ErrBusinessRule,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := BuildPipeline(tokens, 10, []string{"amount", "currency"}, 1000.0)
			err := p.Handle(context.Background(), tt.req)
			if tt.wantErr == nil && err != nil {
				t.Fatalf("esperava sem erro, got %v", err)
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("esperava %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestRateLimit(t *testing.T) {
	p := BuildPipeline(map[string]string{"tok": "bob"}, 2, []string{"x"}, 9999.0)
	ctx := context.Background()
	errs := 0
	for i := 0; i < 5; i++ {
		err := p.Handle(ctx, &Request{Token: "tok", ClientIP: "2.2.2.2", Body: map[string]any{"x": 1}})
		if errors.Is(err, ErrRateLimited) {
			errs++
		}
	}
	if errs == 0 {
		t.Fatalf("esperava rate limit acionar")
	}
}

func TestMainDemo(t *testing.T) {
	main()
}

func TestCtxCancel(t *testing.T) {
	p := BuildPipeline(map[string]string{"tok": "c"}, 10, []string{}, 100.0)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := p.Handle(ctx, &Request{Token: "tok", ClientIP: "3.3.3.3", Body: map[string]any{}})
	if err == nil {
		t.Fatalf("esperava erro de contexto")
	}
}
