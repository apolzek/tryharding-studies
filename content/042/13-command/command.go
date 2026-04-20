package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// Command define uma operacao reversivel (ou nao) que pode ser enfileirada.
type Command interface {
	Name() string
	Execute(ctx context.Context) error
	// Undo pode retornar ErrNotReversible quando nao ha como desfazer.
	Undo(ctx context.Context) error
}

// ErrNotReversible indica que o comando nao suporta undo.
var ErrNotReversible = errors.New("command not reversible")

// --------- Email ---------

type EmailSender interface {
	Send(ctx context.Context, to, subject, body string) error
}

type SendEmailCommand struct {
	Sender  EmailSender
	To      string
	Subject string
	Body    string
}

func (c *SendEmailCommand) Name() string { return "send-email" }
func (c *SendEmailCommand) Execute(ctx context.Context) error {
	return c.Sender.Send(ctx, c.To, c.Subject, c.Body)
}
func (c *SendEmailCommand) Undo(ctx context.Context) error { return ErrNotReversible }

// --------- PDF ---------

type PDFRenderer interface {
	Render(ctx context.Context, docID string) ([]byte, error)
}

type GeneratePDFCommand struct {
	Renderer PDFRenderer
	DocID    string
	Out      *[]byte
}

func (c *GeneratePDFCommand) Name() string { return "generate-pdf" }
func (c *GeneratePDFCommand) Execute(ctx context.Context) error {
	b, err := c.Renderer.Render(ctx, c.DocID)
	if err != nil {
		return err
	}
	if c.Out != nil {
		*c.Out = b
	}
	return nil
}
func (c *GeneratePDFCommand) Undo(ctx context.Context) error { return ErrNotReversible }

// --------- Webhook ---------

type WebhookCaller interface {
	Call(ctx context.Context, url string, payload []byte) error
}

type WebhookCommand struct {
	Caller  WebhookCaller
	URL     string
	Payload []byte
}

func (c *WebhookCommand) Name() string                   { return "webhook" }
func (c *WebhookCommand) Execute(ctx context.Context) error { return c.Caller.Call(ctx, c.URL, c.Payload) }
func (c *WebhookCommand) Undo(ctx context.Context) error   { return ErrNotReversible }

// --------- Conta (reversivel) ---------

type Account struct {
	mu      sync.Mutex
	Balance int64
}

func (a *Account) Debit(v int64) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.Balance < v {
		return errors.New("saldo insuficiente")
	}
	a.Balance -= v
	return nil
}

func (a *Account) Credit(v int64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.Balance += v
}

type DebitCommand struct {
	Acc    *Account
	Amount int64
	done   bool
}

func (c *DebitCommand) Name() string { return "debit" }
func (c *DebitCommand) Execute(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := c.Acc.Debit(c.Amount); err != nil {
		return err
	}
	c.done = true
	return nil
}
func (c *DebitCommand) Undo(ctx context.Context) error {
	if !c.done {
		return nil
	}
	c.Acc.Credit(c.Amount)
	c.done = false
	return nil
}

// --------- Invoker ---------

// Invoker executa comandos e mantem historico para undo.
type Invoker struct {
	mu      sync.Mutex
	history []Command
}

func NewInvoker() *Invoker { return &Invoker{} }

func (i *Invoker) Execute(ctx context.Context, c Command) error {
	if err := c.Execute(ctx); err != nil {
		return fmt.Errorf("%s: %w", c.Name(), err)
	}
	i.mu.Lock()
	i.history = append(i.history, c)
	i.mu.Unlock()
	return nil
}

// UndoLast desfaz o ultimo comando reversivel; ignora nao-reversiveis.
func (i *Invoker) UndoLast(ctx context.Context) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	for n := len(i.history) - 1; n >= 0; n-- {
		c := i.history[n]
		err := c.Undo(ctx)
		if errors.Is(err, ErrNotReversible) {
			continue
		}
		// remove da historia (ate o ponto desfeito)
		i.history = i.history[:n]
		return err
	}
	return errors.New("nada para desfazer")
}

func (i *Invoker) HistorySize() int {
	i.mu.Lock()
	defer i.mu.Unlock()
	return len(i.history)
}
