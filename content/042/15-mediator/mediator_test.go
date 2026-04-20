package main

import (
	"context"
	"errors"
	"testing"
)

func TestJoinLeave(t *testing.T) {
	room := NewAuctionRoom(5)
	a := NewBidder("a")

	tests := []struct {
		name    string
		op      func() error
		wantErr error
	}{
		{"primeiro join ok", func() error { return room.Join(a) }, nil},
		{"join duplicado", func() error { return room.Join(a) }, ErrAlreadyJoined},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.op()
			if tt.wantErr == nil && err != nil {
				t.Fatalf("erro inesperado %v", err)
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("esperava %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestBidFlow(t *testing.T) {
	ctx := context.Background()
	room := NewAuctionRoom(10)
	a := NewBidder("a")
	b := NewBidder("b")
	_ = room.Join(a)
	_ = room.Join(b)

	tests := []struct {
		name    string
		from    string
		value   int64
		wantErr bool
	}{
		{"primeiro lance", "a", 100, false},
		{"lance baixo", "b", 105, true},
		{"lance valido", "b", 120, false},
		{"desconhecido", "x", 200, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := room.Bid(ctx, tt.from, tt.value)
			if tt.wantErr && err == nil {
				t.Fatalf("esperava erro")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}
		})
	}

	room.Close(ctx)
	if err := room.Bid(ctx, "a", 1000); !errors.Is(err, ErrAuctionClosed) {
		t.Fatalf("esperava closed, got %v", err)
	}
}

func TestBroadcast(t *testing.T) {
	ctx := context.Background()
	room := NewAuctionRoom(1)
	a := NewBidder("a")
	b := NewBidder("b")
	c := NewBidder("c")
	_ = room.Join(a)
	_ = room.Join(b)
	_ = room.Join(c)

	if err := room.Broadcast(ctx, "a", "hi"); err != nil {
		t.Fatalf("broadcast: %v", err)
	}
	if len(a.Inbox()) != 0 {
		t.Fatalf("remetente nao deve receber propria msg")
	}
	if len(b.Inbox()) != 1 || len(c.Inbox()) != 1 {
		t.Fatalf("demais participantes devem receber 1 msg cada")
	}
}

func TestBroadcastFechadoEDesconhecido(t *testing.T) {
	ctx := context.Background()
	room := NewAuctionRoom(1)
	a := NewBidder("a")
	_ = room.Join(a)

	if err := room.Broadcast(ctx, "z", "hi"); !errors.Is(err, ErrUnknownParticipant) {
		t.Fatalf("esperava unknown, got %v", err)
	}

	room.Close(ctx)
	if err := room.Broadcast(ctx, "a", "hi"); !errors.Is(err, ErrAuctionClosed) {
		t.Fatalf("esperava closed, got %v", err)
	}
	// Close idempotente.
	room.Close(ctx)
}

func TestLeave(t *testing.T) {
	room := NewAuctionRoom(1)
	a := NewBidder("a")
	_ = room.Join(a)
	room.Leave("a")
	if err := room.Broadcast(context.Background(), "a", "x"); !errors.Is(err, ErrUnknownParticipant) {
		t.Fatalf("esperava unknown apos leave, got %v", err)
	}
}

func TestCtxCancelMediator(t *testing.T) {
	room := NewAuctionRoom(1)
	_ = room.Join(NewBidder("a"))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := room.Broadcast(ctx, "a", "x"); err == nil {
		t.Fatalf("esperava erro de ctx cancelado")
	}
	if err := room.Bid(ctx, "a", 10); err == nil {
		t.Fatalf("esperava erro de ctx em Bid")
	}
}

func TestMainDemo(t *testing.T) {
	main()
}

func TestHighestBid(t *testing.T) {
	room := NewAuctionRoom(5)
	_ = room.Join(NewBidder("a"))
	_ = room.Bid(context.Background(), "a", 50)
	who, val := room.HighestBid()
	if who != "a" || val != 50 {
		t.Fatalf("esperava a/50, got %s/%d", who, val)
	}
}
