package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
)

// Urgency classifica quão urgente é o alerta.
type Urgency int

const (
	UrgencyInfo Urgency = iota
	UrgencyWarning
	UrgencyCritical
)

func (u Urgency) String() string {
	switch u {
	case UrgencyInfo:
		return "INFO"
	case UrgencyWarning:
		return "WARN"
	case UrgencyCritical:
		return "CRIT"
	default:
		return "UNKNOWN"
	}
}

// Channel é a ponte (implementor) — cada canal de envio implementa isso.
type Channel interface {
	Send(ctx context.Context, target, subject, body string) error
}

// Alert é a abstração. Ela não conhece o canal concretamente;
// apenas pede para enviá-la.
type Alert interface {
	Dispatch(ctx context.Context, target string, ch Channel) error
	Urgency() Urgency
}

// BaseAlert contém o estado comum a qualquer alerta.
type BaseAlert struct {
	Title    string
	Message  string
	Priority Urgency
}

func (b BaseAlert) Urgency() Urgency { return b.Priority }

// SystemAlert é um alerta operacional (ex.: métricas, incidentes).
type SystemAlert struct {
	BaseAlert
	Service string
}

func (a SystemAlert) Dispatch(ctx context.Context, target string, ch Channel) error {
	if ch == nil {
		return errors.New("channel required")
	}
	subject := fmt.Sprintf("[%s][%s] %s", a.Priority, a.Service, a.Title)
	return ch.Send(ctx, target, subject, a.Message)
}

// MarketingAlert é um alerta de campanha / uso comercial.
type MarketingAlert struct {
	BaseAlert
	Campaign string
}

func (a MarketingAlert) Dispatch(ctx context.Context, target string, ch Channel) error {
	if ch == nil {
		return errors.New("channel required")
	}
	subject := fmt.Sprintf("[Campanha %s] %s", a.Campaign, a.Title)
	return ch.Send(ctx, target, subject, a.Message)
}

// EmailChannel envia pelo provedor de e-mail. Aqui simulado.
type EmailChannel struct {
	mu   sync.Mutex
	Sent []string
}

func (c *EmailChannel) Send(ctx context.Context, target, subject, body string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !strings.Contains(target, "@") {
		return fmt.Errorf("invalid email: %q", target)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Sent = append(c.Sent, fmt.Sprintf("EMAIL to=%s subj=%q", target, subject))
	_ = body
	return nil
}

// SMSChannel envia SMS.
type SMSChannel struct {
	mu   sync.Mutex
	Sent []string
}

func (c *SMSChannel) Send(ctx context.Context, target, subject, body string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	digits := 0
	for _, r := range target {
		if r >= '0' && r <= '9' {
			digits++
		}
	}
	if digits < 8 {
		return fmt.Errorf("invalid phone: %q", target)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	// SMS colapsa subject+body pra caber em uma mensagem.
	c.Sent = append(c.Sent, fmt.Sprintf("SMS to=%s msg=%q", target, subject+": "+body))
	return nil
}

// SlackChannel envia para um canal do Slack.
type SlackChannel struct {
	mu   sync.Mutex
	Sent []string
}

func (c *SlackChannel) Send(ctx context.Context, target, subject, body string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !strings.HasPrefix(target, "#") {
		return fmt.Errorf("slack target must start with #, got %q", target)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Sent = append(c.Sent, fmt.Sprintf("SLACK to=%s text=%q", target, subject+" — "+body))
	return nil
}
