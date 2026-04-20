package main

import "fmt"

func main() {
	// (2 + 0) * (3 * 1) + -(-7)
	expr := &Binary{
		Op: "+",
		Lhs: &Binary{
			Op:  "*",
			Lhs: &Binary{Op: "+", Lhs: &Number{2}, Rhs: &Number{0}},
			Rhs: &Binary{Op: "*", Lhs: &Number{3}, Rhs: &Number{1}},
		},
		Rhs: &Unary{Op: "-", Expr: &Unary{Op: "-", Expr: &Number{7}}},
	}

	fmt.Println("original :", Print(expr))

	opt := Optimize(expr)
	fmt.Println("otimizada:", Print(opt))

	v, err := Eval(expr)
	if err != nil {
		fmt.Println("erro:", err)
		return
	}
	fmt.Printf("resultado: %.2f\n", v)

	// erro esperado
	bad := &Binary{Op: "/", Lhs: &Number{10}, Rhs: &Number{0}}
	if _, err := Eval(bad); err != nil {
		fmt.Println("falha esperada:", err)
	}
}
