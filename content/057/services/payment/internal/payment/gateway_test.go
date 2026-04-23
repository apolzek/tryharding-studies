package payment

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

type memPub struct{ events []string }

func (p *memPub) Publish(_ context.Context, rk string, _ any) error {
	p.events = append(p.events, rk)
	return nil
}

func TestChargeAlwaysSucceedsWhenGatewayUp(t *testing.T) {
	pub := &memPub{}
	g := NewGateway(pub)
	g.FailureRate = 0.0
	status, _, err := g.Charge(context.Background(), "order-1", 10.0, "key-1")
	assert.NoError(t, err)
	assert.Equal(t, 201, status)
	assert.Equal(t, []string{"payment.confirmed"}, pub.events)
}

func TestChargeGivesUpAfterRetriesAndEmitsFailed(t *testing.T) {
	pub := &memPub{}
	g := NewGateway(pub)
	g.FailureRate = 1.0
	g.MaxRetries = 2
	g.BaseBackoffMs = 1
	status, _, err := g.Charge(context.Background(), "order-2", 5.0, "key-2")
	assert.Error(t, err)
	assert.Equal(t, 502, status)
	assert.Equal(t, []string{"payment.failed"}, pub.events)
}
