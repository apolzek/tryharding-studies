// Package idempotency implements a Redis-backed idempotency cache.
//
// Clients send an Idempotency-Key header. On first request we store
// the response; replays within TTL return the cached response
// unchanged. This is the same pattern that Stripe exposes on its API.
package idempotency

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/go-redis/redis/v8"
)

type Store struct {
	r   *redis.Client
	TTL time.Duration
}

func New(r *redis.Client) *Store {
	return &Store{r: r, TTL: 24 * time.Hour}
}

type cached struct {
	Status int    `json:"s"`
	Body   []byte `json:"b"`
}

var ErrMiss = errors.New("idempotency miss")

func (s *Store) Get(ctx context.Context, key string) (int, []byte, error) {
	if s == nil || s.r == nil || key == "" {
		return 0, nil, ErrMiss
	}
	v, err := s.r.Get(ctx, "idem:"+key).Bytes()
	if errors.Is(err, redis.Nil) {
		return 0, nil, ErrMiss
	}
	if err != nil {
		return 0, nil, err
	}
	var c cached
	if err := json.Unmarshal(v, &c); err != nil {
		return 0, nil, err
	}
	return c.Status, c.Body, nil
}

func (s *Store) Put(ctx context.Context, key string, status int, body []byte) error {
	if s == nil || s.r == nil || key == "" {
		return nil
	}
	b, err := json.Marshal(cached{Status: status, Body: body})
	if err != nil {
		return err
	}
	return s.r.Set(ctx, "idem:"+key, b, s.TTL).Err()
}
