package main

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

// Row é o registro canônico que trafega entre as etapas do pipeline.
type Row struct {
	ID       string
	Customer string
	Amount   float64
}

// ReportSteps são os ganchos que cada formato pode customizar.
type ReportSteps interface {
	Extract() ([]Row, error)
	Transform(rows []Row) ([]Row, error)
	Load(rows []Row) error
	Export(w io.Writer, rows []Row) error
}

// ReportPipeline contém o fluxo fixo: extract -> transform -> load -> export.
type ReportPipeline struct {
	Steps ReportSteps
}

// Run implementa o template method: ordem rígida, etapas plugáveis.
func (p *ReportPipeline) Run(w io.Writer) error {
	if p.Steps == nil {
		return errors.New("pipeline sem steps")
	}
	rows, err := p.Steps.Extract()
	if err != nil {
		return fmt.Errorf("extract: %w", err)
	}
	rows, err = p.Steps.Transform(rows)
	if err != nil {
		return fmt.Errorf("transform: %w", err)
	}
	if err := p.Steps.Load(rows); err != nil {
		return fmt.Errorf("load: %w", err)
	}
	if err := p.Steps.Export(w, rows); err != nil {
		return fmt.Errorf("export: %w", err)
	}
	return nil
}

// baseReport concentra lógica comum reutilizada pelos formatos.
type baseReport struct {
	source []Row
	loaded []Row
}

func (b *baseReport) Extract() ([]Row, error) {
	if b.source == nil {
		return nil, errors.New("source nil")
	}
	out := make([]Row, len(b.source))
	copy(out, b.source)
	return out, nil
}

// Transform comum: remove amount<=0 e ordena por ID para saída estável.
func (b *baseReport) Transform(rows []Row) ([]Row, error) {
	filtered := rows[:0]
	for _, r := range rows {
		if r.Amount > 0 {
			filtered = append(filtered, r)
		}
	}
	sort.SliceStable(filtered, func(i, j int) bool { return filtered[i].ID < filtered[j].ID })
	return filtered, nil
}

func (b *baseReport) Load(rows []Row) error {
	b.loaded = append([]Row(nil), rows...)
	return nil
}

// CSVReport exporta como CSV usando encoding/csv.
type CSVReport struct{ baseReport }

func NewCSVReport(src []Row) *CSVReport { return &CSVReport{baseReport{source: src}} }

func (c *CSVReport) Export(w io.Writer, rows []Row) error {
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{"id", "customer", "amount"}); err != nil {
		return err
	}
	for _, r := range rows {
		if err := cw.Write([]string{r.ID, r.Customer, strconv.FormatFloat(r.Amount, 'f', 2, 64)}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// JSONReport customiza Transform para arredondar amounts e Export para JSON.
type JSONReport struct{ baseReport }

func NewJSONReport(src []Row) *JSONReport { return &JSONReport{baseReport{source: src}} }

// Transform sobrescreve o padrão para também arredondar o valor.
func (j *JSONReport) Transform(rows []Row) ([]Row, error) {
	rows, err := j.baseReport.Transform(rows)
	if err != nil {
		return nil, err
	}
	for i := range rows {
		rows[i].Amount = float64(int(rows[i].Amount*100+0.5)) / 100
	}
	return rows, nil
}

func (j *JSONReport) Export(w io.Writer, rows []Row) error {
	enc := json.NewEncoder(w)
	return enc.Encode(rows)
}

// PDFReport simula um PDF produzindo texto marcado — ilustra customização total.
type PDFReport struct {
	baseReport
	Title string
}

func NewPDFReport(title string, src []Row) *PDFReport {
	return &PDFReport{baseReport: baseReport{source: src}, Title: title}
}

func (p *PDFReport) Export(w io.Writer, rows []Row) error {
	var sb strings.Builder
	sb.WriteString("%PDF-FAKE\n")
	sb.WriteString("Title: " + p.Title + "\n")
	sb.WriteString("----\n")
	for _, r := range rows {
		fmt.Fprintf(&sb, "- %s | %s | %.2f\n", r.ID, r.Customer, r.Amount)
	}
	sb.WriteString("%EOF\n")
	_, err := io.WriteString(w, sb.String())
	return err
}
