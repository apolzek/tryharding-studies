package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"github.com/tryharding/057/customer/internal/events"
	"github.com/tryharding/057/customer/internal/repo"
)

type CustomerService struct {
	repo repo.CustomerRepo
	pub  events.Publisher
}

func NewCustomerService(r repo.CustomerRepo, p events.Publisher) *CustomerService {
	return &CustomerService{repo: r, pub: p}
}

func (s *CustomerService) Create(ctx context.Context, name, email, document string) (*repo.Customer, error) {
	c := &repo.Customer{
		ID:       randomID(),
		Name:     name,
		Email:    email,
		Document: document,
	}
	if err := s.repo.Create(ctx, c); err != nil {
		return nil, err
	}
	if s.pub != nil {
		_ = s.pub.Publish(ctx, "customer.created", map[string]any{
			"id":       c.ID,
			"email":    c.Email,
			"document": c.Document,
		})
	}
	return c, nil
}

func (s *CustomerService) Get(ctx context.Context, id string) (*repo.Customer, error) {
	return s.repo.Get(ctx, id)
}

func (s *CustomerService) List(ctx context.Context) ([]repo.Customer, error) {
	return s.repo.List(ctx)
}

func randomID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
