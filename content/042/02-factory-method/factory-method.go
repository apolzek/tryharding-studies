package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

type TxID string

// PaymentGateway é a abstração que o domínio conhece.
type PaymentGateway interface {
	Charge(ctx context.Context, amountCents int64) (TxID, error)
	Name() string
}

var (
	ErrInvalidAmount     = errors.New("valor inválido")
	ErrUnsupportedGw     = errors.New("gateway não suportado")
	ErrGatewayUnavailable = errors.New("gateway indisponível")
)

// stripeGateway implementa a API da Stripe.
type stripeGateway struct {
	apiKey string
	now    func() time.Time
}

func (s *stripeGateway) Name() string { return "stripe" }
func (s *stripeGateway) Charge(ctx context.Context, amountCents int64) (TxID, error) {
	if amountCents <= 0 {
		return "", ErrInvalidAmount
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return TxID(fmt.Sprintf("stripe_ch_%d", s.now().UnixNano())), nil
}

type paypalGateway struct {
	clientID string
	now      func() time.Time
}

func (p *paypalGateway) Name() string { return "paypal" }
func (p *paypalGateway) Charge(ctx context.Context, amountCents int64) (TxID, error) {
	if amountCents <= 0 {
		return "", ErrInvalidAmount
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return TxID(fmt.Sprintf("paypal_%d", p.now().UnixNano())), nil
}

type pixGateway struct {
	pspToken string
	now      func() time.Time
}

func (p *pixGateway) Name() string { return "pix" }
func (p *pixGateway) Charge(ctx context.Context, amountCents int64) (TxID, error) {
	if amountCents <= 0 {
		return "", ErrInvalidAmount
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return TxID(fmt.Sprintf("pix_e2e_%d", p.now().UnixNano())), nil
}

// Kind identifica o gateway.
type Kind string

const (
	KindStripe Kind = "stripe"
	KindPayPal Kind = "paypal"
	KindPix    Kind = "pix"
)

// Config carrega credenciais possíveis; cada gateway usa só o que precisa.
type Config struct {
	StripeAPIKey   string
	PayPalClientID string
	PixPSPToken    string
	Now            func() time.Time
}

// NewGateway é o factory method: recebe o Kind e devolve a implementação correta.
func NewGateway(kind Kind, cfg Config) (PaymentGateway, error) {
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	switch strings.ToLower(string(kind)) {
	case string(KindStripe):
		if cfg.StripeAPIKey == "" {
			return nil, fmt.Errorf("%w: api key ausente", ErrGatewayUnavailable)
		}
		return &stripeGateway{apiKey: cfg.StripeAPIKey, now: now}, nil
	case string(KindPayPal):
		if cfg.PayPalClientID == "" {
			return nil, fmt.Errorf("%w: client id ausente", ErrGatewayUnavailable)
		}
		return &paypalGateway{clientID: cfg.PayPalClientID, now: now}, nil
	case string(KindPix):
		if cfg.PixPSPToken == "" {
			return nil, fmt.Errorf("%w: token psp ausente", ErrGatewayUnavailable)
		}
		return &pixGateway{pspToken: cfg.PixPSPToken, now: now}, nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedGw, kind)
	}
}
