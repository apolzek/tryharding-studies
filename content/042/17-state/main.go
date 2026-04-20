package main

import "fmt"

func main() {
	o := NewOrder("ORD-1001", 199.90)
	fmt.Printf("pedido %s criado em estado=%s total=%.2f\n", o.ID, o.Status(), o.Total)

	must := func(err error, step string) {
		if err != nil {
			fmt.Printf("  [erro] %s: %v\n", step, err)
			return
		}
		fmt.Printf("  transição ok: %s -> estado=%s\n", step, o.Status())
	}

	must(o.Pay(), "pay")
	must(o.Deliver(), "deliver (inválido)")
	must(o.Ship(), "ship")
	must(o.Cancel(), "cancel (inválido)")
	must(o.Deliver(), "deliver")

	fmt.Println("histórico:", o.History())

	// pedido cancelado antes do envio
	o2 := NewOrder("ORD-1002", 50)
	_ = o2.Pay()
	_ = o2.Cancel()
	fmt.Printf("pedido %s finalizou em estado=%s\n", o2.ID, o2.Status())
}
