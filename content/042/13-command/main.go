package main

import (
	"context"
	"fmt"
)

// Implementacoes fake para demo.
type printEmail struct{}

func (printEmail) Send(ctx context.Context, to, subject, body string) error {
	fmt.Printf("email para=%s subject=%q\n", to, subject)
	return nil
}

type printPDF struct{}

func (printPDF) Render(ctx context.Context, docID string) ([]byte, error) {
	fmt.Printf("pdf renderizado doc=%s\n", docID)
	return []byte("%PDF-fake"), nil
}

type printWebhook struct{}

func (printWebhook) Call(ctx context.Context, url string, payload []byte) error {
	fmt.Printf("webhook url=%s bytes=%d\n", url, len(payload))
	return nil
}

func main() {
	ctx := context.Background()
	inv := NewInvoker()

	acc := &Account{Balance: 1000}

	// Simula uma sequencia de comandos.
	_ = inv.Execute(ctx, &SendEmailCommand{Sender: printEmail{}, To: "a@b.com", Subject: "oi"})
	_ = inv.Execute(ctx, &DebitCommand{Acc: acc, Amount: 200})
	_ = inv.Execute(ctx, &GeneratePDFCommand{Renderer: printPDF{}, DocID: "inv-42"})
	_ = inv.Execute(ctx, &DebitCommand{Acc: acc, Amount: 300})
	_ = inv.Execute(ctx, &WebhookCommand{Caller: printWebhook{}, URL: "https://x.test/hook", Payload: []byte("{}")})

	fmt.Printf("saldo=%d historico=%d\n", acc.Balance, inv.HistorySize())

	// Undo pula webhook (nao reversivel) e estorna o ultimo debit.
	if err := inv.UndoLast(ctx); err != nil {
		fmt.Println("undo erro:", err)
	}
	fmt.Printf("apos undo: saldo=%d historico=%d\n", acc.Balance, inv.HistorySize())
}
