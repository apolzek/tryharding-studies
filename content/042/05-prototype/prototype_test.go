package main

import (
	"reflect"
	"testing"
	"time"
)

func sampleContract() *Contract {
	return &Contract{
		Title:     "T",
		CreatedAt: time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC),
		Parties:   []Party{{Name: "P1", TaxID: "1"}},
		Clauses: []Clause{
			{Title: "C1", Body: "b1", Tags: []string{"a"}, Metrics: map[string]int{"x": 1}},
			{Title: "C2", Body: "b2", Tags: []string{"b"}, Metrics: map[string]int{"y": 2}},
		},
		Metadata: map[string]string{"k": "v"},
	}
}

func TestClauseClone(t *testing.T) {
	t.Run("campos copiados", func(t *testing.T) {
		orig := Clause{Title: "t", Body: "b", Tags: []string{"x"}, Metrics: map[string]int{"p": 1}}
		cp := orig.Clone()
		if !reflect.DeepEqual(orig, cp) {
			t.Fatalf("esperado igual, got %+v", cp)
		}
	})
	t.Run("slices independentes", func(t *testing.T) {
		orig := Clause{Tags: []string{"x"}}
		cp := orig.Clone()
		cp.Tags[0] = "y"
		if orig.Tags[0] == "y" {
			t.Fatal("mutação vazou para o original")
		}
	})
	t.Run("maps independentes", func(t *testing.T) {
		orig := Clause{Metrics: map[string]int{"p": 1}}
		cp := orig.Clone()
		cp.Metrics["p"] = 99
		if orig.Metrics["p"] == 99 {
			t.Fatal("map vazou")
		}
	})
}

func TestContractClone(t *testing.T) {
	t.Run("deep equal mas ponteiros distintos", func(t *testing.T) {
		orig := sampleContract()
		cp := orig.Clone()
		if orig == cp {
			t.Fatal("ponteiros iguais")
		}
		if !reflect.DeepEqual(orig, cp) {
			t.Fatal("conteúdo divergiu")
		}
	})
	t.Run("mutação no clone não afeta original", func(t *testing.T) {
		orig := sampleContract()
		cp := orig.Clone()
		cp.Title = "mudou"
		cp.Parties[0].Name = "outro"
		cp.Clauses[0].Tags[0] = "z"
		cp.Clauses[0].Metrics["x"] = 99
		cp.Metadata["k"] = "w"

		if orig.Title == "mudou" {
			t.Fatal("title vazou")
		}
		if orig.Parties[0].Name == "outro" {
			t.Fatal("parties vazou")
		}
		if orig.Clauses[0].Tags[0] == "z" {
			t.Fatal("clause tags vazou")
		}
		if orig.Clauses[0].Metrics["x"] == 99 {
			t.Fatal("clause metrics vazou")
		}
		if orig.Metadata["k"] == "w" {
			t.Fatal("metadata vazou")
		}
	})
	t.Run("clone de nil", func(t *testing.T) {
		var c *Contract
		if c.Clone() != nil {
			t.Fatal("esperado nil")
		}
	})
	t.Run("append em slice do clone", func(t *testing.T) {
		orig := sampleContract()
		cp := orig.Clone()
		cp.Parties = append(cp.Parties, Party{Name: "novo"})
		if len(orig.Parties) != 1 {
			t.Fatalf("original alterado, len=%d", len(orig.Parties))
		}
	})
}

func TestTemplateRegistry(t *testing.T) {
	t.Run("get devolve clone", func(t *testing.T) {
		r := NewTemplateRegistry()
		r.Register("x", sampleContract())
		a, ok := r.Get("x")
		if !ok {
			t.Fatal("esperado encontrar")
		}
		a.Title = "mudou"
		b, _ := r.Get("x")
		if b.Title == "mudou" {
			t.Fatal("get não clonou")
		}
	})
	t.Run("get inexistente", func(t *testing.T) {
		r := NewTemplateRegistry()
		if _, ok := r.Get("ghost"); ok {
			t.Fatal("esperado false")
		}
	})
}
