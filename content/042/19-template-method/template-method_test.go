package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
)

var seed = []Row{
	{ID: "B", Customer: "Beatriz", Amount: 300.456},
	{ID: "A", Customer: "Alice", Amount: 99.99},
	{ID: "C", Customer: "Carlos", Amount: 0},
	{ID: "D", Customer: "Daniel", Amount: 1500},
}

func TestPipelineFormats(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		build  func() ReportSteps
		verify func(t *testing.T, out string)
	}{
		{
			name:  "csv filtra e ordena",
			build: func() ReportSteps { return NewCSVReport(seed) },
			verify: func(t *testing.T, out string) {
				lines := strings.Split(strings.TrimSpace(out), "\n")
				if len(lines) != 4 {
					t.Fatalf("esperava 4 linhas (header+3), got %d: %q", len(lines), lines)
				}
				if lines[0] != "id,customer,amount" {
					t.Fatalf("header inesperado: %q", lines[0])
				}
				if !strings.HasPrefix(lines[1], "A,Alice,") {
					t.Fatalf("linha 1 deveria começar por A, got %q", lines[1])
				}
				if strings.Contains(out, "Carlos") {
					t.Fatalf("amount zero não devia aparecer")
				}
			},
		},
		{
			name:  "json arredonda amounts",
			build: func() ReportSteps { return NewJSONReport(seed) },
			verify: func(t *testing.T, out string) {
				var got []Row
				if err := json.Unmarshal([]byte(out), &got); err != nil {
					t.Fatalf("json inválido: %v (%q)", err, out)
				}
				if len(got) != 3 {
					t.Fatalf("esperava 3 linhas, got %d", len(got))
				}
				if got[0].ID != "A" {
					t.Fatalf("ordenação falhou: %v", got)
				}
				var found bool
				for _, r := range got {
					if r.ID == "B" && r.Amount == 300.46 {
						found = true
					}
				}
				if !found {
					t.Fatalf("arredondamento falhou: %v", got)
				}
			},
		},
		{
			name:  "pdf gera cabeçalho e corpo",
			build: func() ReportSteps { return NewPDFReport("Q1", seed) },
			verify: func(t *testing.T, out string) {
				if !strings.HasPrefix(out, "%PDF-FAKE") || !strings.HasSuffix(strings.TrimSpace(out), "%EOF") {
					t.Fatalf("envelope pdf ausente: %q", out)
				}
				if !strings.Contains(out, "Title: Q1") {
					t.Fatalf("título ausente")
				}
				if strings.Contains(out, "Carlos") {
					t.Fatalf("amount zero vazou")
				}
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			p := &ReportPipeline{Steps: tc.build()}
			if err := p.Run(&buf); err != nil {
				t.Fatalf("pipeline falhou: %v", err)
			}
			tc.verify(t, buf.String())
		})
	}
}

func TestPipelineLoadPersists(t *testing.T) {
	t.Parallel()
	r := NewCSVReport(seed)
	var buf bytes.Buffer
	if err := (&ReportPipeline{Steps: r}).Run(&buf); err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(r.loaded) != 3 {
		t.Fatalf("load deveria persistir 3 rows, got %d", len(r.loaded))
	}
}

func TestPipelineNilSteps(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	if err := (&ReportPipeline{}).Run(&buf); err == nil {
		t.Fatal("esperava erro com steps nil")
	}
}

var errFake = errors.New("boom")

// stubSteps implementa ReportSteps e falha em uma fase escolhida.
type stubSteps struct{ failOn string }

func (s stubSteps) Extract() ([]Row, error) {
	if s.failOn == "extract" {
		return nil, errFake
	}
	return []Row{{ID: "1", Customer: "x", Amount: 1}}, nil
}
func (s stubSteps) Transform(rows []Row) ([]Row, error) {
	if s.failOn == "transform" {
		return nil, errFake
	}
	return rows, nil
}
func (s stubSteps) Load(rows []Row) error {
	if s.failOn == "load" {
		return errFake
	}
	return nil
}
func (s stubSteps) Export(w io.Writer, rows []Row) error {
	if s.failOn == "export" {
		return errFake
	}
	_, err := w.Write([]byte("ok"))
	return err
}

func TestPipelineErrorPropagation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		failOn string
		prefix string
	}{
		{"extract erra", "extract", "extract"},
		{"transform erra", "transform", "transform"},
		{"load erra", "load", "load"},
		{"export erra", "export", "export"},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			err := (&ReportPipeline{Steps: stubSteps{failOn: tc.failOn}}).Run(&buf)
			if err == nil {
				t.Fatalf("esperava erro")
			}
			if !strings.HasPrefix(err.Error(), tc.prefix+": ") {
				t.Fatalf("erro %q não começa com %q", err.Error(), tc.prefix)
			}
			if !errors.Is(err, errFake) {
				t.Fatalf("erro deveria envolver errFake: %v", err)
			}
		})
	}
}

func TestBaseReportExtractNil(t *testing.T) {
	t.Parallel()
	b := &baseReport{}
	if _, err := b.Extract(); err == nil {
		t.Fatal("esperava erro em source nil")
	}
}

func TestCSVWithNoValidRows(t *testing.T) {
	t.Parallel()
	r := NewCSVReport([]Row{{ID: "A", Amount: 0}, {ID: "B", Amount: -1}})
	var buf bytes.Buffer
	if err := (&ReportPipeline{Steps: r}).Run(&buf); err != nil {
		t.Fatalf("run: %v", err)
	}
	if strings.TrimSpace(buf.String()) != "id,customer,amount" {
		t.Fatalf("saída inesperada: %q", buf.String())
	}
}

func TestMainDoesNotPanic(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("main entrou em pânico: %v", r)
		}
	}()
	main()
}
