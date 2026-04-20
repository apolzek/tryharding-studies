package main

import "fmt"

func main() {
	calc := NewShippingCalculator(
		CorreiosStrategy{},
		TransportadoraStrategy{BaseFee: 25},
		RetiradaStrategy{},
	)

	pkg := Package{WeightKg: 3.2, DistKm: 420, Insured: true, Value: 850}

	for _, k := range []ShippingKind{KindCorreios, KindTransportadora, KindRetirada} {
		price, err := calc.Quote(k, pkg)
		if err != nil {
			fmt.Printf("  [erro] %s: %v\n", k, err)
			continue
		}
		fmt.Printf("modalidade=%-14s preço=R$%7.2f\n", k, price)
	}

	// modalidade inexistente
	if _, err := calc.Quote("drone", pkg); err != nil {
		fmt.Println("falha esperada:", err)
	}
}
