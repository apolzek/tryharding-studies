package main

import "time"

// Clause representa uma cláusula contratual.
type Clause struct {
	Title   string
	Body    string
	Tags    []string
	Metrics map[string]int
}

// Clone devolve uma cópia profunda da cláusula.
func (c Clause) Clone() Clause {
	cp := Clause{
		Title:   c.Title,
		Body:    c.Body,
		Tags:    append([]string(nil), c.Tags...),
		Metrics: make(map[string]int, len(c.Metrics)),
	}
	for k, v := range c.Metrics {
		cp.Metrics[k] = v
	}
	return cp
}

// Party representa uma das partes contratantes.
type Party struct {
	Name    string
	TaxID   string
	Address string
}

// Contract é o documento principal a ser clonado.
type Contract struct {
	Title     string
	CreatedAt time.Time
	Parties   []Party
	Clauses   []Clause
	Metadata  map[string]string
}

// Clone devolve uma cópia profunda do contrato.
func (c *Contract) Clone() *Contract {
	if c == nil {
		return nil
	}
	cp := &Contract{
		Title:     c.Title,
		CreatedAt: c.CreatedAt,
		Parties:   append([]Party(nil), c.Parties...),
		Clauses:   make([]Clause, len(c.Clauses)),
		Metadata:  make(map[string]string, len(c.Metadata)),
	}
	for i, cl := range c.Clauses {
		cp.Clauses[i] = cl.Clone()
	}
	for k, v := range c.Metadata {
		cp.Metadata[k] = v
	}
	return cp
}

// TemplateRegistry armazena protótipos por nome.
type TemplateRegistry struct {
	items map[string]*Contract
}

func NewTemplateRegistry() *TemplateRegistry {
	return &TemplateRegistry{items: map[string]*Contract{}}
}

func (r *TemplateRegistry) Register(name string, c *Contract) {
	r.items[name] = c
}

// Get devolve um clone fresco do template armazenado.
func (r *TemplateRegistry) Get(name string) (*Contract, bool) {
	c, ok := r.items[name]
	if !ok {
		return nil, false
	}
	return c.Clone(), true
}
