package main

import (
	"errors"
	"testing"
)

func TestOrderTransitions(t *testing.T) {
	t.Parallel()

	type step struct {
		action  string
		wantErr bool
		want    Status
	}

	tests := []struct {
		name  string
		steps []step
	}{
		{
			name: "fluxo feliz completo",
			steps: []step{
				{"pay", false, StatusPaid},
				{"ship", false, StatusShipped},
				{"deliver", false, StatusDelivered},
			},
		},
		{
			name: "cancelamento após pagamento",
			steps: []step{
				{"pay", false, StatusPaid},
				{"cancel", false, StatusCanceled},
			},
		},
		{
			name: "cancelamento imediato",
			steps: []step{
				{"cancel", false, StatusCanceled},
			},
		},
		{
			name: "entregar antes de enviar é inválido",
			steps: []step{
				{"pay", false, StatusPaid},
				{"deliver", true, StatusPaid},
			},
		},
		{
			name: "pagar duas vezes é inválido",
			steps: []step{
				{"pay", false, StatusPaid},
				{"pay", true, StatusPaid},
			},
		},
		{
			name: "cancelar após envio é inválido",
			steps: []step{
				{"pay", false, StatusPaid},
				{"ship", false, StatusShipped},
				{"cancel", true, StatusShipped},
			},
		},
		{
			name: "enviar sem pagar é inválido",
			steps: []step{
				{"ship", true, StatusCreated},
			},
		},
		{
			name: "entregar antes de pagar é inválido",
			steps: []step{
				{"deliver", true, StatusCreated},
			},
		},
		{
			name: "pagar ou reenviar depois de shipped é inválido",
			steps: []step{
				{"pay", false, StatusPaid},
				{"ship", false, StatusShipped},
				{"pay", true, StatusShipped},
				{"ship", true, StatusShipped},
			},
		},
		{
			name: "operar pedido entregue é inválido",
			steps: []step{
				{"pay", false, StatusPaid},
				{"ship", false, StatusShipped},
				{"deliver", false, StatusDelivered},
				{"cancel", true, StatusDelivered},
				{"pay", true, StatusDelivered},
				{"ship", true, StatusDelivered},
				{"deliver", true, StatusDelivered},
			},
		},
		{
			name: "operar pedido cancelado é inválido",
			steps: []step{
				{"cancel", false, StatusCanceled},
				{"pay", true, StatusCanceled},
				{"ship", true, StatusCanceled},
				{"deliver", true, StatusCanceled},
				{"cancel", true, StatusCanceled},
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			o := NewOrder("ORD", 10)
			for i, s := range tc.steps {
				var err error
				switch s.action {
				case "pay":
					err = o.Pay()
				case "ship":
					err = o.Ship()
				case "deliver":
					err = o.Deliver()
				case "cancel":
					err = o.Cancel()
				default:
					t.Fatalf("ação desconhecida %q", s.action)
				}
				if s.wantErr {
					if err == nil {
						t.Fatalf("passo %d (%s): esperava erro, estado=%s", i, s.action, o.Status())
					}
					if !errors.Is(err, ErrInvalidTransition) {
						t.Fatalf("passo %d: erro deveria envolver ErrInvalidTransition, got %v", i, err)
					}
				} else if err != nil {
					t.Fatalf("passo %d (%s): erro inesperado: %v", i, s.action, err)
				}
				if o.Status() != s.want {
					t.Fatalf("passo %d: status=%s, quer=%s", i, o.Status(), s.want)
				}
			}
		})
	}
}

func TestMainDoesNotPanic(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("main panic: %v", r)
		}
	}()
	main()
}

func TestHistory(t *testing.T) {
	t.Parallel()
	o := NewOrder("H", 1)
	_ = o.Pay()
	_ = o.Ship()
	hist := o.History()
	if len(hist) != 3 || hist[0] != StatusCreated || hist[2] != StatusShipped {
		t.Fatalf("histórico inesperado: %v", hist)
	}
}
