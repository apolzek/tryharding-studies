package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// Request representa uma requisicao HTTP simplificada.
type Request struct {
	Token   string
	ClientIP string
	Path    string
	Body    map[string]any
	User    string
}

// Handler define a interface de um no da cadeia.
type Handler interface {
	SetNext(Handler) Handler
	Handle(ctx context.Context, r *Request) error
}

// BaseHandler implementa o encadeamento generico.
type BaseHandler struct {
	next Handler
}

func (b *BaseHandler) SetNext(h Handler) Handler {
	b.next = h
	return h
}

func (b *BaseHandler) next_(ctx context.Context, r *Request) error {
	if b.next == nil {
		return nil
	}
	return b.next.Handle(ctx, r)
}

// ErrUnauthorized indica falha de autenticacao.
var ErrUnauthorized = errors.New("unauthorized")

// ErrRateLimited indica excesso de requisicoes.
var ErrRateLimited = errors.New("rate limit exceeded")

// ErrInvalidSchema indica payload invalido.
var ErrInvalidSchema = errors.New("invalid schema")

// ErrBusinessRule indica regra de negocio violada.
var ErrBusinessRule = errors.New("business rule violation")

// AuthHandler valida o token.
type AuthHandler struct {
	BaseHandler
	ValidTokens map[string]string // token -> user
}

func (h *AuthHandler) Handle(ctx context.Context, r *Request) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	user, ok := h.ValidTokens[r.Token]
	if !ok {
		return ErrUnauthorized
	}
	r.User = user
	return h.next_(ctx, r)
}

// RateLimitHandler aplica limite por IP.
type RateLimitHandler struct {
	BaseHandler
	Limit int
	mu    sync.Mutex
	hits  map[string]int
}

func (h *RateLimitHandler) Handle(ctx context.Context, r *Request) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	h.mu.Lock()
	if h.hits == nil {
		h.hits = map[string]int{}
	}
	h.hits[r.ClientIP]++
	count := h.hits[r.ClientIP]
	h.mu.Unlock()
	if count > h.Limit {
		return ErrRateLimited
	}
	return h.next_(ctx, r)
}

// SchemaHandler valida campos obrigatorios do body.
type SchemaHandler struct {
	BaseHandler
	Required []string
}

func (h *SchemaHandler) Handle(ctx context.Context, r *Request) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	for _, f := range h.Required {
		if _, ok := r.Body[f]; !ok {
			return fmt.Errorf("%w: missing %q", ErrInvalidSchema, f)
		}
	}
	return h.next_(ctx, r)
}

// BusinessRuleHandler aplica regras especificas.
type BusinessRuleHandler struct {
	BaseHandler
	MaxAmount float64
}

func (h *BusinessRuleHandler) Handle(ctx context.Context, r *Request) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	amt, _ := r.Body["amount"].(float64)
	if amt > h.MaxAmount {
		return fmt.Errorf("%w: amount above limit", ErrBusinessRule)
	}
	return h.next_(ctx, r)
}

// BuildPipeline monta a cadeia padrao.
func BuildPipeline(tokens map[string]string, rateLimit int, required []string, maxAmount float64) Handler {
	auth := &AuthHandler{ValidTokens: tokens}
	rl := &RateLimitHandler{Limit: rateLimit}
	sc := &SchemaHandler{Required: required}
	br := &BusinessRuleHandler{MaxAmount: maxAmount}
	auth.SetNext(rl).SetNext(sc).SetNext(br)
	return auth
}
