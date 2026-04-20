package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Expr é um nó da AST aritmética. Aceita um visitor para operar sobre si.
type Expr interface {
	Accept(v Visitor) any
}

// Visitor declara um método por tipo de nó, evitando type switch nos clientes.
type Visitor interface {
	VisitNumber(n *Number) any
	VisitBinary(b *Binary) any
	VisitUnary(u *Unary) any
}

// Number — literal numérico.
type Number struct{ Value float64 }

func (n *Number) Accept(v Visitor) any { return v.VisitNumber(n) }

// Binary — operação binária (+, -, *, /).
type Binary struct {
	Op       string
	Lhs, Rhs Expr
}

func (b *Binary) Accept(v Visitor) any { return v.VisitBinary(b) }

// Unary — operação unária (-x).
type Unary struct {
	Op   string
	Expr Expr
}

func (u *Unary) Accept(v Visitor) any { return v.VisitUnary(u) }

// ErrDivByZero é devolvido pelo Evaluator ao encontrar divisão por zero.
var ErrDivByZero = errors.New("divisão por zero")

// Evaluator reduz a AST a um float64 (ou erro).
type Evaluator struct{ Err error }

func (e *Evaluator) VisitNumber(n *Number) any { return n.Value }

func (e *Evaluator) VisitBinary(b *Binary) any {
	if e.Err != nil {
		return 0.0
	}
	l, ok := b.Lhs.Accept(e).(float64)
	if !ok {
		return 0.0
	}
	r, ok := b.Rhs.Accept(e).(float64)
	if !ok {
		return 0.0
	}
	switch b.Op {
	case "+":
		return l + r
	case "-":
		return l - r
	case "*":
		return l * r
	case "/":
		if r == 0 {
			e.Err = ErrDivByZero
			return 0.0
		}
		return l / r
	default:
		e.Err = fmt.Errorf("operador desconhecido %q", b.Op)
		return 0.0
	}
}

func (e *Evaluator) VisitUnary(u *Unary) any {
	if e.Err != nil {
		return 0.0
	}
	v, ok := u.Expr.Accept(e).(float64)
	if !ok {
		return 0.0
	}
	switch u.Op {
	case "-":
		return -v
	case "+":
		return v
	default:
		e.Err = fmt.Errorf("unário desconhecido %q", u.Op)
		return 0.0
	}
}

// Eval conveniente que devolve (valor, erro).
func Eval(e Expr) (float64, error) {
	ev := &Evaluator{}
	out, _ := e.Accept(ev).(float64)
	return out, ev.Err
}

// Printer transforma a AST em string infixada com parênteses.
type Printer struct{}

func (Printer) VisitNumber(n *Number) any {
	return strconv.FormatFloat(n.Value, 'f', -1, 64)
}

func (p Printer) VisitBinary(b *Binary) any {
	lhs, _ := b.Lhs.Accept(p).(string)
	rhs, _ := b.Rhs.Accept(p).(string)
	return "(" + lhs + " " + b.Op + " " + rhs + ")"
}

func (p Printer) VisitUnary(u *Unary) any {
	inner, _ := u.Expr.Accept(p).(string)
	return u.Op + inner
}

// Print devolve a forma textual.
func Print(e Expr) string {
	s, _ := e.Accept(Printer{}).(string)
	return strings.TrimSpace(s)
}

// Optimizer aplica regras triviais e devolve uma nova AST (imutabilidade).
// Regras: x+0=x, 0+x=x, x*1=x, 1*x=x, x*0=0, 0*x=0, --x = x, constant folding.
type Optimizer struct{}

func (Optimizer) VisitNumber(n *Number) any { return &Number{Value: n.Value} }

func (o Optimizer) VisitBinary(b *Binary) any {
	lhs, _ := b.Lhs.Accept(o).(Expr)
	rhs, _ := b.Rhs.Accept(o).(Expr)

	// constant folding: ambos são Number -> reduz
	if ln, okL := lhs.(*Number); okL {
		if rn, okR := rhs.(*Number); okR {
			if v, err := fold(b.Op, ln.Value, rn.Value); err == nil {
				return &Number{Value: v}
			}
		}
	}

	switch b.Op {
	case "+":
		if isZero(lhs) {
			return rhs
		}
		if isZero(rhs) {
			return lhs
		}
	case "*":
		if isZero(lhs) || isZero(rhs) {
			return &Number{Value: 0}
		}
		if isOne(lhs) {
			return rhs
		}
		if isOne(rhs) {
			return lhs
		}
	}
	return &Binary{Op: b.Op, Lhs: lhs, Rhs: rhs}
}

func (o Optimizer) VisitUnary(u *Unary) any {
	inner, _ := u.Expr.Accept(o).(Expr)
	// --x = x
	if u.Op == "-" {
		if inner2, ok := inner.(*Unary); ok && inner2.Op == "-" {
			return inner2.Expr
		}
	}
	// +x = x
	if u.Op == "+" {
		return inner
	}
	return &Unary{Op: u.Op, Expr: inner}
}

// Optimize aplica o visitor e devolve a nova expressão.
func Optimize(e Expr) Expr {
	out, _ := e.Accept(Optimizer{}).(Expr)
	return out
}

// helpers internos

func isZero(e Expr) bool {
	n, ok := e.(*Number)
	return ok && n.Value == 0
}

func isOne(e Expr) bool {
	n, ok := e.(*Number)
	return ok && n.Value == 1
}

func fold(op string, l, r float64) (float64, error) {
	switch op {
	case "+":
		return l + r, nil
	case "-":
		return l - r, nil
	case "*":
		return l * r, nil
	case "/":
		if r == 0 {
			return 0, ErrDivByZero
		}
		return l / r, nil
	}
	return 0, fmt.Errorf("op inválido %q", op)
}
