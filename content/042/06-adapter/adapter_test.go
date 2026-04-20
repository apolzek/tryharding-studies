package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestLegacyToModernAdapter_Charge(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		client      LegacyPaymentClient
		req         PaymentRequest
		wantApprove bool
		wantErr     bool
	}{
		{
			name:        "happy path approves",
			client:      &FakeLegacyClient{},
			req:         PaymentRequest{OrderID: "O1", AmountBR: 10.5, Customer: "A"},
			wantApprove: true,
		},
		{
			name:        "legacy denies payment",
			client:      &FakeLegacyClient{Deny: true},
			req:         PaymentRequest{OrderID: "O2", AmountBR: 10.5, Customer: "B"},
			wantApprove: false,
		},
		{
			name:    "missing order id",
			client:  &FakeLegacyClient{},
			req:     PaymentRequest{AmountBR: 10, Customer: "C"},
			wantErr: true,
		},
		{
			name:    "zero amount rejected",
			client:  &FakeLegacyClient{},
			req:     PaymentRequest{OrderID: "O3", Customer: "C"},
			wantErr: true,
		},
		{
			name:    "empty customer rejected",
			client:  &FakeLegacyClient{},
			req:     PaymentRequest{OrderID: "O4", AmountBR: 12, Customer: "   "},
			wantErr: true,
		},
		{
			name:    "transport failure bubbles up",
			client:  &FakeLegacyClient{FailTransport: true},
			req:     PaymentRequest{OrderID: "O5", AmountBR: 1, Customer: "D"},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			adp := NewLegacyToModernAdapter(tc.client)
			resp, err := adp.Charge(ctx, tc.req)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (resp=%+v)", resp)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.Approved != tc.wantApprove {
				t.Fatalf("approved=%v want %v (msg=%s)", resp.Approved, tc.wantApprove, resp.Message)
			}
			if tc.wantApprove && resp.AuthCode == "" {
				t.Fatalf("expected auth code when approved")
			}
		})
	}
}

// stubLegacy permite forçar erros específicos.
type stubLegacy struct {
	err  error
	resp []byte
}

func (s *stubLegacy) SendXML(ctx context.Context, payload []byte) ([]byte, error) {
	return s.resp, s.err
}

func TestLegacyToModernAdapter_InvalidResponse(t *testing.T) {
	t.Parallel()

	adp := NewLegacyToModernAdapter(&stubLegacy{resp: []byte("<<<not xml")})
	_, err := adp.Charge(context.Background(), PaymentRequest{OrderID: "X", AmountBR: 1, Customer: "Y"})
	if err == nil {
		t.Fatalf("expected unmarshal error")
	}
}

func TestRunDemoOutputs(t *testing.T) {
	t.Parallel()
	var buf strings.Builder
	run(context.Background(), &buf)
	out := buf.String()
	if !strings.Contains(out, "approved=true") {
		t.Errorf("expected success output, got %q", out)
	}
	if !strings.Contains(out, "approved=false") {
		t.Errorf("expected denial output, got %q", out)
	}
}

func TestLegacyToModernAdapter_ContextCancelled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	adp := NewLegacyToModernAdapter(&FakeLegacyClient{})
	_, err := adp.Charge(ctx, PaymentRequest{OrderID: "X", AmountBR: 1, Customer: "Y"})
	if !errors.Is(err, context.Canceled) && err == nil {
		t.Fatalf("expected cancellation error, got %v", err)
	}
}
