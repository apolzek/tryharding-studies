package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	run(ctx, os.Stdout)
}

func run(ctx context.Context, w io.Writer) {
	legacy := &FakeLegacyClient{}
	var gateway ModernPaymentGateway = NewLegacyToModernAdapter(legacy)

	resp, err := gateway.Charge(ctx, PaymentRequest{
		OrderID:  "ORD-1001",
		AmountBR: 199.90,
		Customer: "Ana Costa",
	})
	if err != nil {
		fmt.Fprintln(w, "erro:", err)
		return
	}
	fmt.Fprintf(w, "approved=%v auth=%s msg=%s\n", resp.Approved, resp.AuthCode, resp.Message)

	legacyDeny := &FakeLegacyClient{Deny: true}
	gateway = NewLegacyToModernAdapter(legacyDeny)
	resp, err = gateway.Charge(ctx, PaymentRequest{
		OrderID:  "ORD-1002",
		AmountBR: 50.00,
		Customer: "Bruno",
	})
	if err != nil {
		fmt.Fprintln(w, "erro:", err)
		return
	}
	fmt.Fprintf(w, "approved=%v msg=%s\n", resp.Approved, resp.Message)
}
