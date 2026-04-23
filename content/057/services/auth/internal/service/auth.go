package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/tryharding/057/auth/internal/repo"

	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

type AuthService struct {
	users  repo.UserRepo
	redis  *redis.Client
	secret []byte
}

func NewAuthService(u repo.UserRepo, r *redis.Client, secret string) *AuthService {
	return &AuthService{users: u, redis: r, secret: []byte(secret)}
}

func (s *AuthService) Register(ctx context.Context, email, password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	id := randomID()
	u := &repo.User{ID: id, Email: email, Password: string(hash)}
	if err := s.users.Create(ctx, u); err != nil {
		return "", err
	}
	return id, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, error) {
	u, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		return "", err
	}
	if u == nil {
		return "", ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)); err != nil {
		return "", ErrInvalidCredentials
	}
	token, err := s.issueToken(u.ID, u.Email)
	if err != nil {
		return "", err
	}
	if s.redis != nil {
		_ = s.redis.Set(ctx, "session:"+u.ID, token, 1*time.Hour).Err()
	}
	return token, nil
}

func (s *AuthService) Validate(tokenStr string) (string, error) {
	tok, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return s.secret, nil
	})
	if err != nil || !tok.Valid {
		return "", ErrInvalidCredentials
	}
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return "", ErrInvalidCredentials
	}
	sub, _ := claims["sub"].(string)
	return sub, nil
}

func (s *AuthService) issueToken(sub, email string) (string, error) {
	claims := jwt.MapClaims{
		"sub":   sub,
		"email": email,
		"exp":   time.Now().Add(1 * time.Hour).Unix(),
		"iat":   time.Now().Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString(s.secret)
}

func randomID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
