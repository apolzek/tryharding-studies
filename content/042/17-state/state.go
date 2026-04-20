package main

import (
	"errors"
	"fmt"
)

// Status representa o estado do ciclo de vida do pedido.
type Status string

const (
	StatusCreated   Status = "created"
	StatusPaid      Status = "paid"
	StatusShipped   Status = "shipped"
	StatusDelivered Status = "delivered"
	StatusCanceled  Status = "canceled"
)

// ErrInvalidTransition indica transição proibida na máquina de estados.
var ErrInvalidTransition = errors.New("transição de estado inválida")

// OrderState define o contrato de cada estado concreto.
type OrderState interface {
	Name() Status
	Pay(o *Order) error
	Ship(o *Order) error
	Deliver(o *Order) error
	Cancel(o *Order) error
}

// Order é o contexto que mantém o estado atual.
type Order struct {
	ID    string
	Total float64
	state OrderState
	log   []Status
}

// NewOrder cria um pedido iniciando em Created.
func NewOrder(id string, total float64) *Order {
	o := &Order{ID: id, Total: total}
	o.setState(&createdState{})
	return o
}

func (o *Order) setState(s OrderState) {
	o.state = s
	o.log = append(o.log, s.Name())
}

// Status retorna o estado atual.
func (o *Order) Status() Status { return o.state.Name() }

// History devolve a trilha de estados percorridos.
func (o *Order) History() []Status { return append([]Status(nil), o.log...) }

// Ações disparam transições delegando ao estado corrente.
func (o *Order) Pay() error     { return o.state.Pay(o) }
func (o *Order) Ship() error    { return o.state.Ship(o) }
func (o *Order) Deliver() error { return o.state.Deliver(o) }
func (o *Order) Cancel() error  { return o.state.Cancel(o) }

// createdState — pedido criado, aguarda pagamento.
type createdState struct{}

func (createdState) Name() Status           { return StatusCreated }
func (createdState) Pay(o *Order) error     { o.setState(&paidState{}); return nil }
func (createdState) Ship(*Order) error      { return fmt.Errorf("%w: created->shipped", ErrInvalidTransition) }
func (createdState) Deliver(*Order) error   { return fmt.Errorf("%w: created->delivered", ErrInvalidTransition) }
func (createdState) Cancel(o *Order) error  { o.setState(&canceledState{}); return nil }

// paidState — pedido pago, pode ser despachado ou cancelado.
type paidState struct{}

func (paidState) Name() Status          { return StatusPaid }
func (paidState) Pay(*Order) error      { return fmt.Errorf("%w: já pago", ErrInvalidTransition) }
func (paidState) Ship(o *Order) error   { o.setState(&shippedState{}); return nil }
func (paidState) Deliver(*Order) error  { return fmt.Errorf("%w: paid->delivered", ErrInvalidTransition) }
func (paidState) Cancel(o *Order) error { o.setState(&canceledState{}); return nil }

// shippedState — em trânsito.
type shippedState struct{}

func (shippedState) Name() Status             { return StatusShipped }
func (shippedState) Pay(*Order) error         { return fmt.Errorf("%w: shipped->paid", ErrInvalidTransition) }
func (shippedState) Ship(*Order) error        { return fmt.Errorf("%w: já despachado", ErrInvalidTransition) }
func (shippedState) Deliver(o *Order) error   { o.setState(&deliveredState{}); return nil }
func (shippedState) Cancel(*Order) error      { return fmt.Errorf("%w: já despachado", ErrInvalidTransition) }

// deliveredState — estado final feliz.
type deliveredState struct{}

func (deliveredState) Name() Status            { return StatusDelivered }
func (deliveredState) Pay(*Order) error        { return fmt.Errorf("%w: já entregue", ErrInvalidTransition) }
func (deliveredState) Ship(*Order) error       { return fmt.Errorf("%w: já entregue", ErrInvalidTransition) }
func (deliveredState) Deliver(*Order) error    { return fmt.Errorf("%w: já entregue", ErrInvalidTransition) }
func (deliveredState) Cancel(*Order) error     { return fmt.Errorf("%w: já entregue", ErrInvalidTransition) }

// canceledState — estado final triste.
type canceledState struct{}

func (canceledState) Name() Status           { return StatusCanceled }
func (canceledState) Pay(*Order) error       { return fmt.Errorf("%w: cancelado", ErrInvalidTransition) }
func (canceledState) Ship(*Order) error      { return fmt.Errorf("%w: cancelado", ErrInvalidTransition) }
func (canceledState) Deliver(*Order) error   { return fmt.Errorf("%w: cancelado", ErrInvalidTransition) }
func (canceledState) Cancel(*Order) error    { return fmt.Errorf("%w: já cancelado", ErrInvalidTransition) }
