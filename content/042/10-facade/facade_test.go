package main

import (
	"context"
	"testing"
)

func baseOrder() Order {
	return Order{
		ID:            "O1",
		CustomerEmail: "c@x.com",
		SKU:           "SKU",
		Quantity:      1,
		TotalBR:       100,
	}
}

func TestCheckout_PlaceOrder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		modify       func(*Checkout)
		order        Order
		wantErr      bool
		wantNotified bool
	}{
		{
			name:         "happy path",
			order:        baseOrder(),
			wantNotified: true,
		},
		{
			name:    "out of stock",
			modify:  func(c *Checkout) { c.Inventory = &InMemoryInventory{Stock: map[string]int{"SKU": 0}} },
			order:   baseOrder(),
			wantErr: true,
		},
		{
			name:    "payment declined rolls back",
			modify:  func(c *Checkout) { c.Payments = &FakePayments{Fail: true} },
			order:   baseOrder(),
			wantErr: true,
		},
		{
			name:    "shipping failure refunds and releases",
			modify:  func(c *Checkout) { c.Shipping = &FakeShipping{Fail: true} },
			order:   baseOrder(),
			wantErr: true,
		},
		{
			name:    "invalid order rejected",
			order:   Order{ID: "", SKU: "x", Quantity: 1, TotalBR: 1, CustomerEmail: "a@b"},
			wantErr: true,
		},
		{
			name:    "zero quantity rejected",
			order:   Order{ID: "X", SKU: "x", Quantity: 0, TotalBR: 1, CustomerEmail: "a@b"},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			inv := &InMemoryInventory{Stock: map[string]int{"SKU": 5}}
			pay := &FakePayments{}
			ship := &FakeShipping{}
			notif := &FakeNotifier{}
			c := &Checkout{Inventory: inv, Payments: pay, Shipping: ship, Notifier: notif}
			if tc.modify != nil {
				tc.modify(c)
			}

			// Valor antes, para verificar rollback de estoque
			stockBefore := 0
			if im, ok := c.Inventory.(*InMemoryInventory); ok {
				stockBefore = im.Stock["SKU"]
			}

			_, err := c.PlaceOrder(context.Background(), tc.order)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				// Se subsistema falhou depois da reserva, estoque deve ter sido devolvido.
				if im, ok := c.Inventory.(*InMemoryInventory); ok {
					if im.Stock["SKU"] != stockBefore {
						t.Errorf("stock not rolled back: before=%d after=%d", stockBefore, im.Stock["SKU"])
					}
				}
				if tc.wantNotified {
					t.Errorf("should not notify on failure")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantNotified && len(notif.Sent) != 1 {
				t.Errorf("notifier calls=%d want 1", len(notif.Sent))
			}
		})
	}
}

func TestCheckout_ShippingFailureRefunds(t *testing.T) {
	t.Parallel()
	inv := &InMemoryInventory{Stock: map[string]int{"SKU": 3}}
	pay := &FakePayments{}
	c := &Checkout{Inventory: inv, Payments: pay, Shipping: &FakeShipping{Fail: true}, Notifier: &FakeNotifier{}}
	_, err := c.PlaceOrder(context.Background(), baseOrder())
	if err == nil {
		t.Fatal("expected error")
	}
	if _, ok := pay.Authorized("AUTH-O1"); ok {
		t.Error("payment should have been refunded")
	}
	if inv.Stock["SKU"] != 3 {
		t.Errorf("inventory not restored: %d", inv.Stock["SKU"])
	}
}

func TestCheckout_ContextCanceled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c := &Checkout{
		Inventory: &InMemoryInventory{Stock: map[string]int{"SKU": 5}},
		Payments:  &FakePayments{},
		Shipping:  &FakeShipping{},
		Notifier:  &FakeNotifier{},
	}
	_, err := c.PlaceOrder(ctx, baseOrder())
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}
