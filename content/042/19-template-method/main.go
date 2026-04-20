package main

import (
	"fmt"
	"os"
)

func main() {
	data := []Row{
		{ID: "B", Customer: "Beatriz", Amount: 300.456},
		{ID: "A", Customer: "Alice", Amount: 99.99},
		{ID: "C", Customer: "Carlos", Amount: 0}, // deve ser filtrado
		{ID: "D", Customer: "Daniel", Amount: 1500},
	}

	fmt.Println("---- CSV ----")
	csvReport := NewCSVReport(data)
	if err := (&ReportPipeline{Steps: csvReport}).Run(os.Stdout); err != nil {
		fmt.Println("erro:", err)
	}

	fmt.Println("---- JSON ----")
	jsonReport := NewJSONReport(data)
	if err := (&ReportPipeline{Steps: jsonReport}).Run(os.Stdout); err != nil {
		fmt.Println("erro:", err)
	}

	fmt.Println("---- PDF ----")
	pdfReport := NewPDFReport("Relatório Mensal", data)
	if err := (&ReportPipeline{Steps: pdfReport}).Run(os.Stdout); err != nil {
		fmt.Println("erro:", err)
	}
}
