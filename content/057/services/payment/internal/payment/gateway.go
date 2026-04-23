// Package payment simulates a payment gateway.
//
// - 10% of requests fail randomly (we retry with exponential backoff
//   before finally giving up and emitting payment.failed).
// - Successful charges publish payment.confirmed and include a
//   transaction id the order service records.
package payment

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"math"
	mrand "math/rand"
	"time"

	"github.com/tryharding/057/payment/internal/events"
)

type Gateway struct {
	pub           events.Publisher
	FailureRate   float64
	MaxRetries    int
	BaseBackoffMs int
}

func NewGateway(pub events.Publisher) *Gateway {
	return &Gateway{
		pub:           pub,
		FailureRate:   0.10,
		MaxRetries:    3,
		BaseBackoffMs: 100,
	}
}

type ChargeResult struct {
	OrderID       string  `json:"order_id"`
	TransactionID string  `json:"transaction_id"`
	Amount        float64 `json:"amount"`
	Status        string  `json:"status"`
	Attempts      int     `json:"attempts"`
}

var ErrGatewayDown = errors.New("gateway transient error")

// Charge attempts to bill. Retries on transient errors with
// exponential backoff and jitter, then emits the outcome.
func (g *Gateway) Charge(ctx context.Context, orderID string, amount float64, idempotencyKey string) (int, any, error) {
	var attempts int
	for attempt := 0; attempt <= g.MaxRetries; attempt++ {
		attempts = attempt + 1
		if ok := g.simulateGateway(); ok {
			res := ChargeResult{
				OrderID:       orderID,
				TransactionID: "tx_" + randomHex(10),
				Amount:        amount,
				Status:        "confirmed",
				Attempts:      attempts,
			}
			if g.pub != nil {
				_ = g.pub.Publish(ctx, "payment.confirmed", res)
			}
			return 201, res, nil
		}
		// exponential backoff with jitter
		wait := time.Duration(float64(g.BaseBackoffMs)*math.Pow(2, float64(attempt))+float64(mrand.Intn(50))) * time.Millisecond
		select {
		case <-ctx.Done():
			return 0, nil, ctx.Err()
		case <-time.After(wait):
		}
	}

	res := ChargeResult{
		OrderID:  orderID,
		Amount:   amount,
		Status:   "failed",
		Attempts: attempts,
	}
	if g.pub != nil {
		_ = g.pub.Publish(ctx, "payment.failed", map[string]any{
			"order_id": orderID,
			"reason":   "gateway unavailable after retries",
			"attempts": attempts,
		})
	}
	return 502, res, ErrGatewayDown
}

func (g *Gateway) simulateGateway() bool {
	return mrand.Float64() >= g.FailureRate
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
