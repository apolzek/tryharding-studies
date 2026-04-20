package main

import (
	"errors"
	"math"
	"testing"
)

func TestStrategies(t *testing.T) {
	t.Parallel()

	calc := NewShippingCalculator(
		CorreiosStrategy{},
		TransportadoraStrategy{BaseFee: 20},
		RetiradaStrategy{},
	)

	tests := []struct {
		name     string
		kind     ShippingKind
		pkg      Package
		want     float64
		wantErr  bool
		errIs    error
	}{
		{
			name: "correios peso e distância padrão",
			kind: KindCorreios,
			pkg:  Package{WeightKg: 2, DistKm: 100},
			want: round2(8 + 2*3.5 + 100*0.12),
		},
		{
			name: "correios abaixo do piso aplica piso",
			kind: KindCorreios,
			pkg:  Package{WeightKg: 0.1, DistKm: 1},
			want: 12,
		},
		{
			name: "correios com seguro soma 1% do valor",
			kind: KindCorreios,
			pkg:  Package{WeightKg: 5, DistKm: 200, Insured: true, Value: 1000},
			want: round2(8 + 5*3.5 + 200*0.12 + 1000*0.01),
		},
		{
			name:    "correios peso inválido erra",
			kind:    KindCorreios,
			pkg:     Package{WeightKg: 0, DistKm: 10},
			wantErr: true,
		},
		{
			name: "transportadora padrão",
			kind: KindTransportadora,
			pkg:  Package{WeightKg: 10, DistKm: 500},
			want: round2(20 + 10*2.1 + 500*0.28),
		},
		{
			name: "transportadora com seguro",
			kind: KindTransportadora,
			pkg:  Package{WeightKg: 8, DistKm: 300, Insured: true, Value: 2000},
			want: round2(20 + 8*2.1 + 300*0.28 + 2000*0.015),
		},
		{
			name:    "transportadora peso inválido erra",
			kind:    KindTransportadora,
			pkg:     Package{WeightKg: -1},
			wantErr: true,
		},
		{
			name: "retirada sempre zero",
			kind: KindRetirada,
			pkg:  Package{WeightKg: 100, DistKm: 10000, Insured: true, Value: 999999},
			want: 0,
		},
		{
			name:    "modalidade desconhecida",
			kind:    ShippingKind("teleporte"),
			pkg:     Package{WeightKg: 1, DistKm: 1},
			wantErr: true,
			errIs:   ErrUnknownStrategy,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := calc.Quote(tc.kind, tc.pkg)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("esperava erro, got preço=%.2f", got)
				}
				if tc.errIs != nil && !errors.Is(err, tc.errIs) {
					t.Fatalf("erro deveria envolver %v, got %v", tc.errIs, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}
			if math.Abs(got-tc.want) > 0.01 {
				t.Fatalf("preço=%.2f, quer=%.2f", got, tc.want)
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

func TestStrategyNames(t *testing.T) {
	t.Parallel()
	cases := []struct {
		s    ShippingStrategy
		want ShippingKind
	}{
		{CorreiosStrategy{}, KindCorreios},
		{TransportadoraStrategy{}, KindTransportadora},
		{RetiradaStrategy{}, KindRetirada},
	}
	for _, c := range cases {
		if c.s.Name() != c.want {
			t.Fatalf("Name()=%s, quer=%s", c.s.Name(), c.want)
		}
	}
}
