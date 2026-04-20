package main

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// PaymentRequest é o contrato moderno (REST/JSON) usado pelo sistema atual.
type PaymentRequest struct {
	OrderID  string
	AmountBR float64 // valor em BRL
	Customer string
}

// PaymentResponse representa a resposta no formato moderno.
type PaymentResponse struct {
	AuthCode string
	Approved bool
	Message  string
}

// ModernPaymentGateway é a interface esperada pelo domínio.
type ModernPaymentGateway interface {
	Charge(ctx context.Context, req PaymentRequest) (PaymentResponse, error)
}

// LegacySOAPPayload é o envelope legado (SOAP/XML-like).
type LegacySOAPPayload struct {
	XMLName xml.Name `xml:"PaymentEnvelope"`
	TxnID   string   `xml:"TransactionID"`
	// O sistema legado trabalha em centavos como string.
	AmountCents string `xml:"AmountCents"`
	PayerName   string `xml:"PayerName"`
}

// LegacySOAPResponse é a resposta crua do sistema legado.
type LegacySOAPResponse struct {
	XMLName   xml.Name `xml:"PaymentResult"`
	Status    string   `xml:"Status"` // "OK" / "DENIED" / "FAIL"
	Code      string   `xml:"AuthorizationCode"`
	ErrorText string   `xml:"ErrorText,omitempty"`
}

// LegacyPaymentClient é o cliente legado. Interface diferente da moderna.
type LegacyPaymentClient interface {
	SendXML(ctx context.Context, payload []byte) ([]byte, error)
}

// FakeLegacyClient simula o cliente SOAP; em produção seria um HTTP client
// falando com um endpoint legado.
type FakeLegacyClient struct {
	// Se Deny for true, todo pagamento é negado.
	Deny bool
	// FailTransport simula falha de rede/transporte.
	FailTransport bool
}

func (c *FakeLegacyClient) SendXML(ctx context.Context, payload []byte) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if c.FailTransport {
		return nil, errors.New("legacy transport failure")
	}

	var req LegacySOAPPayload
	if err := xml.Unmarshal(payload, &req); err != nil {
		return nil, fmt.Errorf("invalid xml: %w", err)
	}
	if req.AmountCents == "" || req.TxnID == "" {
		resp := LegacySOAPResponse{Status: "FAIL", ErrorText: "missing fields"}
		return xml.Marshal(resp)
	}
	if c.Deny {
		resp := LegacySOAPResponse{Status: "DENIED", ErrorText: "insufficient funds"}
		return xml.Marshal(resp)
	}
	resp := LegacySOAPResponse{Status: "OK", Code: "AUTH-" + req.TxnID}
	return xml.Marshal(resp)
}

// LegacyToModernAdapter adapta LegacyPaymentClient à interface moderna.
type LegacyToModernAdapter struct {
	Legacy LegacyPaymentClient
}

func NewLegacyToModernAdapter(legacy LegacyPaymentClient) *LegacyToModernAdapter {
	return &LegacyToModernAdapter{Legacy: legacy}
}

func (a *LegacyToModernAdapter) Charge(ctx context.Context, req PaymentRequest) (PaymentResponse, error) {
	if req.OrderID == "" {
		return PaymentResponse{}, errors.New("orderID required")
	}
	if req.AmountBR <= 0 {
		return PaymentResponse{}, errors.New("amount must be positive")
	}
	if strings.TrimSpace(req.Customer) == "" {
		return PaymentResponse{}, errors.New("customer required")
	}

	cents := strconv.FormatInt(int64(req.AmountBR*100+0.5), 10)
	payload := LegacySOAPPayload{
		TxnID:       req.OrderID,
		AmountCents: cents,
		PayerName:   req.Customer,
	}
	raw, err := xml.Marshal(payload)
	if err != nil {
		return PaymentResponse{}, fmt.Errorf("marshal legacy payload: %w", err)
	}

	out, err := a.Legacy.SendXML(ctx, raw)
	if err != nil {
		return PaymentResponse{}, fmt.Errorf("legacy call: %w", err)
	}

	var resp LegacySOAPResponse
	if err := xml.Unmarshal(out, &resp); err != nil {
		return PaymentResponse{}, fmt.Errorf("unmarshal legacy response: %w", err)
	}

	switch resp.Status {
	case "OK":
		return PaymentResponse{AuthCode: resp.Code, Approved: true, Message: "authorized"}, nil
	case "DENIED":
		return PaymentResponse{Approved: false, Message: resp.ErrorText}, nil
	default:
		return PaymentResponse{}, fmt.Errorf("legacy failure: %s", resp.ErrorText)
	}
}
