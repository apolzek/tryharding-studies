package main

import (
	"context"
	"fmt"
	"time"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	checkout := &Checkout{
		Inventory: &InMemoryInventory{Stock: map[string]int{"SKU-1": 5}},
		Payments:  &FakePayments{},
		Shipping:  &FakeShipping{},
		Notifier:  &FakeNotifier{},
	}

	receipt, err := checkout.PlaceOrder(ctx, Order{
		ID:            "ORD-1",
		CustomerEmail: "ana@example.com",
		SKU:           "SKU-1",
		Quantity:      2,
		TotalBR:       199.90,
	})
	if err != nil {
		fmt.Println("erro:", err)
		return
	}
	fmt.Printf("ok: %+v\n", receipt)

	// Falha simulada de pagamento
	failingCheckout := &Checkout{
		Inventory: &InMemoryInventory{Stock: map[string]int{"SKU-2": 3}},
		Payments:  &FakePayments{Fail: true},
		Shipping:  &FakeShipping{},
		Notifier:  &FakeNotifier{},
	}
	_, err = failingCheckout.PlaceOrder(ctx, Order{
		ID:            "ORD-2",
		CustomerEmail: "bruno@example.com",
		SKU:           "SKU-2",
		Quantity:      1,
		TotalBR:       50,
	})
	fmt.Println("pedido ORD-2 falhou como esperado:", err)
}
