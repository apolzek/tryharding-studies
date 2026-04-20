package main

import (
	"errors"
	"math"
	"testing"
)

func TestEvaluator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		expr    Expr
		want    float64
		wantErr bool
		errIs   error
	}{
		{"número literal", &Number{42}, 42, false, nil},
		{"soma simples", &Binary{Op: "+", Lhs: &Number{2}, Rhs: &Number{3}}, 5, false, nil},
		{"subtração", &Binary{Op: "-", Lhs: &Number{10}, Rhs: &Number{4}}, 6, false, nil},
		{"multiplicação", &Binary{Op: "*", Lhs: &Number{6}, Rhs: &Number{7}}, 42, false, nil},
		{"divisão", &Binary{Op: "/", Lhs: &Number{20}, Rhs: &Number{4}}, 5, false, nil},
		{"unário negativo", &Unary{Op: "-", Expr: &Number{3}}, -3, false, nil},
		{"unário positivo", &Unary{Op: "+", Expr: &Number{3}}, 3, false, nil},
		{
			name: "expressão composta",
			expr: &Binary{
				Op:  "+",
				Lhs: &Binary{Op: "*", Lhs: &Number{2}, Rhs: &Number{3}},
				Rhs: &Unary{Op: "-", Expr: &Number{1}},
			},
			want: 5,
		},
		{
			name:    "divisão por zero",
			expr:    &Binary{Op: "/", Lhs: &Number{1}, Rhs: &Number{0}},
			wantErr: true, errIs: ErrDivByZero,
		},
		{
			name:    "operador desconhecido",
			expr:    &Binary{Op: "%", Lhs: &Number{1}, Rhs: &Number{2}},
			wantErr: true,
		},
		{
			name:    "unário desconhecido",
			expr:    &Unary{Op: "!", Expr: &Number{1}},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := Eval(tc.expr)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("esperava erro, got %v", got)
				}
				if tc.errIs != nil && !errors.Is(err, tc.errIs) {
					t.Fatalf("errIs=%v, got %v", tc.errIs, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}
			if math.Abs(got-tc.want) > 1e-9 {
				t.Fatalf("got=%v, quer=%v", got, tc.want)
			}
		})
	}
}

func TestMainDoesNotPanic(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("main panic: %v", r)
		}
	}()
	main()
}

func TestPrinter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		expr Expr
		want string
	}{
		{"número", &Number{3}, "3"},
		{"binário", &Binary{Op: "+", Lhs: &Number{1}, Rhs: &Number{2}}, "(1 + 2)"},
		{"unário", &Unary{Op: "-", Expr: &Number{5}}, "-5"},
		{
			name: "aninhado",
			expr: &Binary{Op: "*", Lhs: &Binary{Op: "+", Lhs: &Number{1}, Rhs: &Number{2}}, Rhs: &Number{3}},
			want: "((1 + 2) * 3)",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := Print(tc.expr); got != tc.want {
				t.Fatalf("got=%q, quer=%q", got, tc.want)
			}
		})
	}
}

func TestOptimizer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		expr Expr
		want string // Print do resultado
	}{
		{"x+0=x", &Binary{Op: "+", Lhs: &Number{5}, Rhs: &Number{0}}, "5"},
		{"0+x=x", &Binary{Op: "+", Lhs: &Number{0}, Rhs: &Number{5}}, "5"},
		{"x*1=x", &Binary{Op: "*", Lhs: &Number{7}, Rhs: &Number{1}}, "7"},
		{"1*x=x", &Binary{Op: "*", Lhs: &Number{1}, Rhs: &Number{7}}, "7"},
		{"x*0=0", &Binary{Op: "*", Lhs: &Number{99}, Rhs: &Number{0}}, "0"},
		{"0*x=0", &Binary{Op: "*", Lhs: &Number{0}, Rhs: &Number{99}}, "0"},
		{"--x=x", &Unary{Op: "-", Expr: &Unary{Op: "-", Expr: &Number{3}}}, "3"},
		{"+x=x", &Unary{Op: "+", Expr: &Number{3}}, "3"},
		{"constant folding", &Binary{Op: "+", Lhs: &Number{2}, Rhs: &Number{3}}, "5"},
		{
			name: "recursão em ambos os lados",
			expr: &Binary{
				Op:  "*",
				Lhs: &Binary{Op: "+", Lhs: &Number{3}, Rhs: &Number{0}},
				Rhs: &Binary{Op: "*", Lhs: &Number{1}, Rhs: &Number{4}},
			},
			want: "12",
		},
		{
			name: "não mexe no que não reconhece",
			expr: &Binary{Op: "-", Lhs: &Number{5}, Rhs: &Number{0}},
			want: "5", // fold aritmético aplica
		},
		{
			name: "divisão por zero não faz folding",
			expr: &Binary{Op: "/", Lhs: &Number{1}, Rhs: &Number{0}},
			want: "(1 / 0)",
		},
		{
			name: "operador não reconhecido não faz folding",
			expr: &Binary{Op: "%", Lhs: &Number{1}, Rhs: &Number{2}},
			want: "(1 % 2)",
		},
		{
			name: "unário desconhecido preserva",
			expr: &Unary{Op: "!", Expr: &Number{1}},
			want: "!1",
		},
		{
			name: "binário sem identidade preserva",
			expr: &Binary{Op: "+", Lhs: &Number{3}, Rhs: &Binary{Op: "+", Lhs: &Number{0}, Rhs: &Number{4}}},
			want: "7",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Print(Optimize(tc.expr))
			if got != tc.want {
				t.Fatalf("got=%q, quer=%q", got, tc.want)
			}
		})
	}
}
