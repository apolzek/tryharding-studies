package main

import (
	"context"
	"fmt"
)

func runPipeline(ctx context.Context, f CloudFactory) {
	s := f.NewStorage()
	q := f.NewQueue()

	if err := s.Put(ctx, "invoices/2024-01.pdf", []byte("PDF-BYTES")); err != nil {
		fmt.Printf("put erro: %v\n", err)
		return
	}
	data, err := s.Get(ctx, "invoices/2024-01.pdf")
	if err != nil {
		fmt.Printf("get erro: %v\n", err)
		return
	}
	id, err := q.Publish(ctx, "billing.invoice.created", data)
	if err != nil {
		fmt.Printf("publish erro: %v\n", err)
		return
	}
	fmt.Printf("[%s] storage=%s queue=%s msg=%s\n", f.Region(), s.Provider(), q.Provider(), id)
}

func main() {
	ctx := context.Background()
	for _, cfg := range []struct {
		prov   Provider
		region string
	}{
		{ProviderAWS, "us-east-1"},
		{ProviderGCP, "southamerica-east1"},
	} {
		f, err := NewCloudFactory(cfg.prov, cfg.region)
		if err != nil {
			fmt.Printf("falha criando %s: %v\n", cfg.prov, err)
			continue
		}
		runPipeline(ctx, f)
	}

	if _, err := NewCloudFactory("azure", "br-south"); err != nil {
		fmt.Printf("provedor não suportado -> %v\n", err)
	}
}
