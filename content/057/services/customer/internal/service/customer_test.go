package service

import (
	"context"
	"testing"

	"github.com/tryharding/057/customer/internal/repo"

	"github.com/stretchr/testify/assert"
)

type memRepo struct{ items map[string]*repo.Customer }

func (m *memRepo) Create(_ context.Context, c *repo.Customer) error { m.items[c.ID] = c; return nil }
func (m *memRepo) Get(_ context.Context, id string) (*repo.Customer, error) {
	return m.items[id], nil
}
func (m *memRepo) List(_ context.Context) ([]repo.Customer, error) {
	out := []repo.Customer{}
	for _, v := range m.items {
		out = append(out, *v)
	}
	return out, nil
}

type memPub struct{ events []string }

func (p *memPub) Publish(_ context.Context, rk string, _ any) error {
	p.events = append(p.events, rk)
	return nil
}

func TestCreateCustomerPublishesEvent(t *testing.T) {
	r := &memRepo{items: map[string]*repo.Customer{}}
	p := &memPub{}
	s := NewCustomerService(r, p)
	c, err := s.Create(context.Background(), "Alice", "a@b.com", "1234")
	assert.NoError(t, err)
	assert.NotEmpty(t, c.ID)
	assert.Equal(t, []string{"customer.created"}, p.events)
}
