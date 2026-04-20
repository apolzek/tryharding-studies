package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func fixedClock() func() time.Time {
	t := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	return func() time.Time { return t }
}

func TestNewGateway(t *testing.T) {
	tests := []struct {
		name     string
		kind     Kind
		cfg      Config
		wantErr  error
		wantName string
	}{
		{"stripe ok", KindStripe, Config{StripeAPIKey: "k", Now: fixedClock()}, nil, "stripe"},
		{"paypal ok", KindPayPal, Config{PayPalClientID: "c", Now: fixedClock()}, nil, "paypal"},
		{"pix ok", KindPix, Config{PixPSPToken: "t", Now: fixedClock()}, nil, "pix"},
		{"stripe sem key", KindStripe, Config{}, ErrGatewayUnavailable, ""},
		{"paypal sem id", KindPayPal, Config{}, ErrGatewayUnavailable, ""},
		{"pix sem token", KindPix, Config{}, ErrGatewayUnavailable, ""},
		{"kind inválido", "crypto", Config{}, ErrUnsupportedGw, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gw, err := NewGateway(tc.kind, tc.cfg)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("esperado %v, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}
			if gw.Name() != tc.wantName {
				t.Fatalf("nome esperado %s, got %s", tc.wantName, gw.Name())
			}
		})
	}
}

func TestCharge(t *testing.T) {
	ctx := context.Background()
	cfg := Config{StripeAPIKey: "k", PayPalClientID: "c", PixPSPToken: "t", Now: fixedClock()}

	cases := []struct {
		name       string
		kind       Kind
		amount     int64
		wantPrefix string
		wantErr    error
	}{
		{"stripe happy", KindStripe, 1000, "stripe_ch_", nil},
		{"paypal happy", KindPayPal, 2500, "paypal_", nil},
		{"pix happy", KindPix, 999, "pix_e2e_", nil},
		{"stripe zero", KindStripe, 0, "", ErrInvalidAmount},
		{"paypal negativo", KindPayPal, -1, "", ErrInvalidAmount},
		{"pix negativo", KindPix, -100, "", ErrInvalidAmount},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gw, err := NewGateway(tc.kind, cfg)
			if err != nil {
				t.Fatal(err)
			}
			tx, err := gw.Charge(ctx, tc.amount)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("esperado %v, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}
			if !strings.HasPrefix(string(tx), tc.wantPrefix) {
				t.Fatalf("tx %q deveria começar com %q", tx, tc.wantPrefix)
			}
		})
	}
}

func TestChargeContextCanceled(t *testing.T) {
	cfg := Config{StripeAPIKey: "k", Now: fixedClock()}
	gw, err := NewGateway(KindStripe, cfg)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := gw.Charge(ctx, 100); err == nil {
		t.Fatal("esperado erro de contexto cancelado")
	}
}

func TestCaseInsensitiveKind(t *testing.T) {
	cfg := Config{StripeAPIKey: "k", Now: fixedClock()}
	gw, err := NewGateway("STRIPE", cfg)
	if err != nil {
		t.Fatal(err)
	}
	if gw.Name() != "stripe" {
		t.Fatalf("esperado stripe, got %s", gw.Name())
	}
}
