package main

import (
	"context"
	"fmt"
	"time"
)

func main() {
	cfg := Config{
		StripeAPIKey:   "sk_test_123",
		PayPalClientID: "ppl_client_abc",
		PixPSPToken:    "pix_token_xyz",
	}

	kinds := []Kind{KindStripe, KindPayPal, KindPix}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	for _, k := range kinds {
		gw, err := NewGateway(k, cfg)
		if err != nil {
			fmt.Printf("[%s] falha: %v\n", k, err)
			continue
		}
		tx, err := gw.Charge(ctx, 4990)
		if err != nil {
			fmt.Printf("[%s] erro no charge: %v\n", gw.Name(), err)
			continue
		}
		fmt.Printf("[%s] tx=%s\n", gw.Name(), tx)
	}

	if _, err := NewGateway("bitcoin", cfg); err != nil {
		fmt.Printf("gateway desconhecido -> %v\n", err)
	}
}
