package service

import (
	"context"
	"testing"

	"github.com/tryharding/057/auth/internal/repo"

	"github.com/stretchr/testify/assert"
)

type memRepo struct{ users map[string]*repo.User }

func (m *memRepo) Create(_ context.Context, u *repo.User) error {
	m.users[u.Email] = u
	return nil
}
func (m *memRepo) FindByEmail(_ context.Context, email string) (*repo.User, error) {
	return m.users[email], nil
}

func TestRegisterAndLogin(t *testing.T) {
	m := &memRepo{users: map[string]*repo.User{}}
	s := NewAuthService(m, nil, "testsecret")

	_, err := s.Register(context.Background(), "a@b.com", "pw12345")
	assert.NoError(t, err)

	tok, err := s.Login(context.Background(), "a@b.com", "pw12345")
	assert.NoError(t, err)
	assert.NotEmpty(t, tok)

	sub, err := s.Validate(tok)
	assert.NoError(t, err)
	assert.NotEmpty(t, sub)
}

func TestLoginInvalid(t *testing.T) {
	m := &memRepo{users: map[string]*repo.User{}}
	s := NewAuthService(m, nil, "testsecret")
	_, err := s.Login(context.Background(), "x@y.com", "whatever")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}
