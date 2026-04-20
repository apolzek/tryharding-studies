package main

import (
	"errors"
	"fmt"
	"strings"
)

// OrgNode é o componente comum. Tanto Employee (folha) quanto Department
// (composição) implementam essa interface.
type OrgNode interface {
	Name() string
	TotalSalary() float64
	Headcount() int
	// Print imprime a árvore indentada a partir deste nó.
	Print(indent int) string
}

// Employee é a folha da árvore.
type Employee struct {
	FullName string
	Role     string
	Salary   float64
}

func (e *Employee) Name() string         { return e.FullName }
func (e *Employee) TotalSalary() float64 { return e.Salary }
func (e *Employee) Headcount() int       { return 1 }
func (e *Employee) Print(indent int) string {
	return fmt.Sprintf("%s- %s (%s) R$%.2f", strings.Repeat("  ", indent), e.FullName, e.Role, e.Salary)
}

// Department é o nó composto. Contém outros OrgNodes.
type Department struct {
	Title    string
	children []OrgNode
}

func NewDepartment(title string) *Department {
	return &Department{Title: title}
}

func (d *Department) Name() string { return d.Title }

func (d *Department) Add(nodes ...OrgNode) error {
	for _, n := range nodes {
		if n == nil {
			return errors.New("nil node not allowed")
		}
		if n == d {
			return errors.New("department cannot contain itself")
		}
	}
	d.children = append(d.children, nodes...)
	return nil
}

// Children retorna cópia defensiva dos filhos.
func (d *Department) Children() []OrgNode {
	out := make([]OrgNode, len(d.children))
	copy(out, d.children)
	return out
}

func (d *Department) TotalSalary() float64 {
	var total float64
	for _, c := range d.children {
		total += c.TotalSalary()
	}
	return total
}

func (d *Department) Headcount() int {
	count := 0
	for _, c := range d.children {
		count += c.Headcount()
	}
	return count
}

func (d *Department) Print(indent int) string {
	var sb strings.Builder
	sb.WriteString(strings.Repeat("  ", indent))
	sb.WriteString("+ ")
	sb.WriteString(d.Title)
	sb.WriteString(fmt.Sprintf(" (%d pessoas, R$%.2f)", d.Headcount(), d.TotalSalary()))
	for _, c := range d.children {
		sb.WriteString("\n")
		sb.WriteString(c.Print(indent + 1))
	}
	return sb.String()
}

// AverageSalary percorre a árvore. Retorna 0 quando não há empregados.
func AverageSalary(n OrgNode) float64 {
	if n == nil || n.Headcount() == 0 {
		return 0
	}
	return n.TotalSalary() / float64(n.Headcount())
}
