package main

import (
	"fmt"
	"time"
)

func buildNDATemplate() *Contract {
	return &Contract{
		Title:     "NDA Padrão",
		CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Parties:   []Party{{Name: "Empresa X", TaxID: "00.000.000/0001-00"}},
		Clauses: []Clause{
			{Title: "Confidencialidade", Body: "As Partes comprometem-se...", Tags: []string{"sigilo"}, Metrics: map[string]int{"peso": 3}},
			{Title: "Vigência", Body: "Este contrato vigorará por 24 meses...", Tags: []string{"prazo"}, Metrics: map[string]int{"peso": 2}},
		},
		Metadata: map[string]string{"lang": "pt-BR", "version": "1.0"},
	}
}

func main() {
	registry := NewTemplateRegistry()
	registry.Register("nda", buildNDATemplate())

	// Gera dois contratos a partir do mesmo template.
	a, _ := registry.Get("nda")
	a.Parties = append(a.Parties, Party{Name: "Cliente Alpha", TaxID: "11.111.111/0001-11"})
	a.Metadata["deal_id"] = "deal-001"
	a.Clauses[0].Tags = append(a.Clauses[0].Tags, "alpha-customização")

	b, _ := registry.Get("nda")
	b.Parties = append(b.Parties, Party{Name: "Cliente Beta", TaxID: "22.222.222/0001-22"})
	b.Metadata["deal_id"] = "deal-002"

	fmt.Printf("A parties=%d tags[0]=%v deal=%s\n", len(a.Parties), a.Clauses[0].Tags, a.Metadata["deal_id"])
	fmt.Printf("B parties=%d tags[0]=%v deal=%s\n", len(b.Parties), b.Clauses[0].Tags, b.Metadata["deal_id"])

	// O template original permanece intacto.
	orig, _ := registry.Get("nda")
	fmt.Printf("Template original parties=%d tags[0]=%v metadata=%v\n",
		len(orig.Parties), orig.Clauses[0].Tags, orig.Metadata)
}
