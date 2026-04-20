package main

import (
	"context"
	"errors"
	"testing"
)

type fakeEmail struct{ fail bool }

func (f fakeEmail) Send(ctx context.Context, to, subject, body string) error {
	if f.fail {
		return errors.New("smtp down")
	}
	return nil
}

type fakePDF struct{}

func (fakePDF) Render(ctx context.Context, id string) ([]byte, error) { return []byte("x"), nil }

type fakeHook struct{}

func (fakeHook) Call(ctx context.Context, url string, p []byte) error { return nil }

func TestInvokerExecute(t *testing.T) {
	ctx := context.Background()
	acc := &Account{Balance: 500}

	tests := []struct {
		name    string
		cmd     Command
		wantErr bool
	}{
		{"email ok", &SendEmailCommand{Sender: fakeEmail{}, To: "x"}, false},
		{"email fail", &SendEmailCommand{Sender: fakeEmail{fail: true}, To: "x"}, true},
		{"pdf", &GeneratePDFCommand{Renderer: fakePDF{}, DocID: "z"}, false},
		{"webhook", &WebhookCommand{Caller: fakeHook{}, URL: "u"}, false},
		{"debit ok", &DebitCommand{Acc: acc, Amount: 100}, false},
		{"debit fail", &DebitCommand{Acc: acc, Amount: 99999}, true},
	}

	inv := NewInvoker()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := inv.Execute(ctx, tt.cmd)
			if tt.wantErr && err == nil {
				t.Fatalf("esperava erro")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}
		})
	}
}

func TestUndoReversible(t *testing.T) {
	ctx := context.Background()
	acc := &Account{Balance: 1000}
	inv := NewInvoker()

	_ = inv.Execute(ctx, &DebitCommand{Acc: acc, Amount: 200})
	_ = inv.Execute(ctx, &WebhookCommand{Caller: fakeHook{}, URL: "u"})
	_ = inv.Execute(ctx, &DebitCommand{Acc: acc, Amount: 100})

	if acc.Balance != 700 {
		t.Fatalf("saldo esperado 700, got %d", acc.Balance)
	}

	// Ultimo debit = 100 deve ser estornado.
	if err := inv.UndoLast(ctx); err != nil {
		t.Fatalf("undo: %v", err)
	}
	if acc.Balance != 800 {
		t.Fatalf("apos undo, saldo esperado 800, got %d", acc.Balance)
	}

	// Proximo undo: webhook nao reversivel, pula; estorna debit 200.
	if err := inv.UndoLast(ctx); err != nil {
		t.Fatalf("undo 2: %v", err)
	}
	if acc.Balance != 1000 {
		t.Fatalf("apos undo 2, saldo esperado 1000, got %d", acc.Balance)
	}
}

func TestUndoSemHistorico(t *testing.T) {
	inv := NewInvoker()
	if err := inv.UndoLast(context.Background()); err == nil {
		t.Fatalf("esperava erro de historia vazia")
	}
}

func TestCommandNames(t *testing.T) {
	cases := []Command{
		&SendEmailCommand{Sender: fakeEmail{}},
		&GeneratePDFCommand{Renderer: fakePDF{}},
		&WebhookCommand{Caller: fakeHook{}},
		&DebitCommand{Acc: &Account{}},
	}
	for _, c := range cases {
		t.Run(c.Name(), func(t *testing.T) {
			if c.Name() == "" {
				t.Fatalf("nome vazio")
			}
		})
	}
}

func TestPDFOut(t *testing.T) {
	var out []byte
	cmd := &GeneratePDFCommand{Renderer: fakePDF{}, DocID: "d", Out: &out}
	if err := cmd.Execute(context.Background()); err != nil {
		t.Fatal(err)
	}
	if string(out) != "x" {
		t.Fatalf("out nao preenchido: %q", string(out))
	}
}

func TestUndoNaoReversiveis(t *testing.T) {
	ctx := context.Background()
	cmds := []Command{
		&SendEmailCommand{Sender: fakeEmail{}},
		&GeneratePDFCommand{Renderer: fakePDF{}},
		&WebhookCommand{Caller: fakeHook{}},
	}
	for _, c := range cmds {
		if err := c.Undo(ctx); !errors.Is(err, ErrNotReversible) {
			t.Fatalf("%s: esperava ErrNotReversible, got %v", c.Name(), err)
		}
	}
}

func TestDebitUndoSemExecucao(t *testing.T) {
	d := &DebitCommand{Acc: &Account{Balance: 10}, Amount: 5}
	// Undo sem Execute: no-op.
	if err := d.Undo(context.Background()); err != nil {
		t.Fatalf("undo no-op falhou: %v", err)
	}
	if d.Acc.Balance != 10 {
		t.Fatalf("saldo nao deveria mudar")
	}
}

func TestHistorySize(t *testing.T) {
	inv := NewInvoker()
	if inv.HistorySize() != 0 {
		t.Fatalf("esperava 0")
	}
	_ = inv.Execute(context.Background(), &WebhookCommand{Caller: fakeHook{}, URL: "u"})
	if inv.HistorySize() != 1 {
		t.Fatalf("esperava 1")
	}
}

func TestMainDemo(t *testing.T) {
	// Exercita main.go para cobertura do pacote.
	main()
}
