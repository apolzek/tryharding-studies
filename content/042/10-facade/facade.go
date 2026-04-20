package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// Order é o pedido que o cliente quer colocar.
type Order struct {
	ID        string
	CustomerEmail string
	SKU       string
	Quantity  int
	TotalBR   float64
}

// Receipt é o retorno da fachada.
type Receipt struct {
	OrderID   string
	AuthCode  string
	Tracking  string
}

// --- Subsistemas ---

// InventoryService gerencia estoque.
type InventoryService interface {
	Reserve(ctx context.Context, sku string, qty int) error
	Release(ctx context.Context, sku string, qty int) error
}

// PaymentService processa cobrança.
type PaymentService interface {
	Charge(ctx context.Context, orderID string, amount float64) (authCode string, err error)
	Refund(ctx context.Context, authCode string) error
}

// ShippingService despacha.
type ShippingService interface {
	Ship(ctx context.Context, orderID, sku string) (tracking string, err error)
}

// NotificationService avisa o cliente.
type NotificationService interface {
	NotifySuccess(ctx context.Context, email, orderID, tracking string) error
}

// --- Facade ---

// Checkout é a fachada. Esconde a orquestração: reservar estoque,
// cobrar, despachar e notificar. Compensa passos em caso de falha.
type Checkout struct {
	Inventory   InventoryService
	Payments    PaymentService
	Shipping    ShippingService
	Notifier    NotificationService
}

// PlaceOrder é a única entrada exposta aos clientes da fachada.
func (c *Checkout) PlaceOrder(ctx context.Context, order Order) (Receipt, error) {
	if err := validate(order); err != nil {
		return Receipt{}, err
	}

	if err := c.Inventory.Reserve(ctx, order.SKU, order.Quantity); err != nil {
		return Receipt{}, fmt.Errorf("reserve inventory: %w", err)
	}

	auth, err := c.Payments.Charge(ctx, order.ID, order.TotalBR)
	if err != nil {
		_ = c.Inventory.Release(ctx, order.SKU, order.Quantity)
		return Receipt{}, fmt.Errorf("charge payment: %w", err)
	}

	tracking, err := c.Shipping.Ship(ctx, order.ID, order.SKU)
	if err != nil {
		_ = c.Payments.Refund(ctx, auth)
		_ = c.Inventory.Release(ctx, order.SKU, order.Quantity)
		return Receipt{}, fmt.Errorf("ship: %w", err)
	}

	// Notificação é best-effort: não reverte o pedido em caso de falha.
	_ = c.Notifier.NotifySuccess(ctx, order.CustomerEmail, order.ID, tracking)

	return Receipt{OrderID: order.ID, AuthCode: auth, Tracking: tracking}, nil
}

func validate(o Order) error {
	if o.ID == "" {
		return errors.New("order id required")
	}
	if o.SKU == "" {
		return errors.New("sku required")
	}
	if o.Quantity <= 0 {
		return errors.New("quantity must be > 0")
	}
	if o.TotalBR <= 0 {
		return errors.New("total must be > 0")
	}
	if o.CustomerEmail == "" {
		return errors.New("customer email required")
	}
	return nil
}

// --- Implementações in-memory para o exemplo ---

type InMemoryInventory struct {
	mu    sync.Mutex
	Stock map[string]int
}

func (i *InMemoryInventory) Reserve(ctx context.Context, sku string, qty int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.Stock[sku] < qty {
		return fmt.Errorf("out of stock for %s", sku)
	}
	i.Stock[sku] -= qty
	return nil
}

func (i *InMemoryInventory) Release(ctx context.Context, sku string, qty int) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.Stock[sku] += qty
	return nil
}

type FakePayments struct {
	Fail  bool
	mu    sync.Mutex
	auths map[string]float64
}

func (p *FakePayments) Charge(ctx context.Context, orderID string, amount float64) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if p.Fail {
		return "", errors.New("payment declined")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.auths == nil {
		p.auths = map[string]float64{}
	}
	auth := "AUTH-" + orderID
	p.auths[auth] = amount
	return auth, nil
}

func (p *FakePayments) Refund(ctx context.Context, auth string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.auths, auth)
	return nil
}

func (p *FakePayments) Authorized(auth string) (float64, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	v, ok := p.auths[auth]
	return v, ok
}

type FakeShipping struct {
	Fail bool
}

func (s *FakeShipping) Ship(ctx context.Context, orderID, sku string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if s.Fail {
		return "", errors.New("carrier unavailable")
	}
	return "TRK-" + orderID, nil
}

type FakeNotifier struct {
	mu   sync.Mutex
	Sent []string
}

func (n *FakeNotifier) NotifySuccess(ctx context.Context, email, orderID, tracking string) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.Sent = append(n.Sent, fmt.Sprintf("to=%s order=%s tracking=%s", email, orderID, tracking))
	return nil
}
