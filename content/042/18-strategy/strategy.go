package main

import (
	"errors"
	"fmt"
	"math"
)

// ShippingKind identifica a modalidade de entrega escolhida pelo cliente.
type ShippingKind string

const (
	KindCorreios      ShippingKind = "correios"
	KindTransportadora ShippingKind = "transportadora"
	KindRetirada      ShippingKind = "retirada"
)

// Package resume os dados necessários para cotar o frete.
type Package struct {
	WeightKg float64
	DistKm   float64
	Insured  bool
	Value    float64
}

// ErrUnknownStrategy é retornado quando a modalidade não está registrada.
var ErrUnknownStrategy = errors.New("estratégia de frete desconhecida")

// ShippingStrategy é o contrato dos algoritmos intercambiáveis.
type ShippingStrategy interface {
	Name() ShippingKind
	Quote(p Package) (float64, error)
}

// Correios — tarifa baseada em peso e distância com piso.
type CorreiosStrategy struct{}

func (CorreiosStrategy) Name() ShippingKind { return KindCorreios }
func (CorreiosStrategy) Quote(p Package) (float64, error) {
	if p.WeightKg <= 0 {
		return 0, errors.New("peso inválido")
	}
	price := 8.0 + p.WeightKg*3.5 + p.DistKm*0.12
	if p.Insured {
		price += p.Value * 0.01
	}
	return round2(math.Max(price, 12)), nil
}

// Transportadora — frete dedicado, mais caro porém sem piso fixo.
type TransportadoraStrategy struct {
	BaseFee float64
}

func (t TransportadoraStrategy) Name() ShippingKind { return KindTransportadora }
func (t TransportadoraStrategy) Quote(p Package) (float64, error) {
	if p.WeightKg <= 0 {
		return 0, errors.New("peso inválido")
	}
	price := t.BaseFee + p.WeightKg*2.1 + p.DistKm*0.28
	if p.Insured {
		price += p.Value * 0.015
	}
	return round2(price), nil
}

// Retirada — cliente busca na loja, custo zero.
type RetiradaStrategy struct{}

func (RetiradaStrategy) Name() ShippingKind         { return KindRetirada }
func (RetiradaStrategy) Quote(_ Package) (float64, error) { return 0, nil }

// ShippingCalculator seleciona e delega para a estratégia correta em runtime.
type ShippingCalculator struct {
	strategies map[ShippingKind]ShippingStrategy
}

// NewShippingCalculator recebe as estratégias disponíveis.
func NewShippingCalculator(ss ...ShippingStrategy) *ShippingCalculator {
	m := make(map[ShippingKind]ShippingStrategy, len(ss))
	for _, s := range ss {
		m[s.Name()] = s
	}
	return &ShippingCalculator{strategies: m}
}

// Quote resolve a estratégia por tipo e calcula o frete.
func (c *ShippingCalculator) Quote(kind ShippingKind, p Package) (float64, error) {
	s, ok := c.strategies[kind]
	if !ok {
		return 0, fmt.Errorf("%w: %s", ErrUnknownStrategy, kind)
	}
	return s.Quote(p)
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }
