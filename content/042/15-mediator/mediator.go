package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// Participant e um ator da sala de leilao.
type Participant interface {
	ID() string
	Receive(msg Message)
}

// Message trafega mensagens e lances.
type Message struct {
	From string
	Kind string // "chat" | "bid" | "sold"
	Text string
	Bid  int64
}

// ErrAlreadyJoined indica registro duplicado.
var ErrAlreadyJoined = errors.New("participant ja registrado")

// ErrUnknownParticipant indica ator nao registrado.
var ErrUnknownParticipant = errors.New("participant desconhecido")

// ErrAuctionClosed indica leilao fechado.
var ErrAuctionClosed = errors.New("leilao fechado")

// AuctionRoom implementa o mediador.
type AuctionRoom struct {
	mu           sync.Mutex
	participants map[string]Participant
	highestBid   int64
	highestBy    string
	minIncrement int64
	closed       bool
}

// NewAuctionRoom cria a sala com incremento minimo obrigatorio por lance.
func NewAuctionRoom(minIncrement int64) *AuctionRoom {
	return &AuctionRoom{
		participants: map[string]Participant{},
		minIncrement: minIncrement,
	}
}

// Join registra um participante.
func (a *AuctionRoom) Join(p Participant) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, ok := a.participants[p.ID()]; ok {
		return ErrAlreadyJoined
	}
	a.participants[p.ID()] = p
	return nil
}

// Leave remove um participante.
func (a *AuctionRoom) Leave(id string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.participants, id)
}

// Broadcast encaminha chat entre todos exceto o remetente.
func (a *AuctionRoom) Broadcast(ctx context.Context, from, text string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	a.mu.Lock()
	if a.closed {
		a.mu.Unlock()
		return ErrAuctionClosed
	}
	if _, ok := a.participants[from]; !ok {
		a.mu.Unlock()
		return ErrUnknownParticipant
	}
	targets := make([]Participant, 0, len(a.participants)-1)
	for id, p := range a.participants {
		if id == from {
			continue
		}
		targets = append(targets, p)
	}
	a.mu.Unlock()

	for _, t := range targets {
		t.Receive(Message{From: from, Kind: "chat", Text: text})
	}
	return nil
}

// Bid registra um lance se respeitar incremento minimo.
func (a *AuctionRoom) Bid(ctx context.Context, from string, value int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	a.mu.Lock()
	if a.closed {
		a.mu.Unlock()
		return ErrAuctionClosed
	}
	if _, ok := a.participants[from]; !ok {
		a.mu.Unlock()
		return ErrUnknownParticipant
	}
	if value < a.highestBid+a.minIncrement {
		a.mu.Unlock()
		return fmt.Errorf("lance %d abaixo do minimo %d", value, a.highestBid+a.minIncrement)
	}
	a.highestBid = value
	a.highestBy = from
	targets := make([]Participant, 0, len(a.participants))
	for _, p := range a.participants {
		targets = append(targets, p)
	}
	a.mu.Unlock()

	for _, t := range targets {
		t.Receive(Message{From: from, Kind: "bid", Bid: value})
	}
	return nil
}

// Close encerra e anuncia vencedor.
func (a *AuctionRoom) Close(ctx context.Context) {
	a.mu.Lock()
	if a.closed {
		a.mu.Unlock()
		return
	}
	a.closed = true
	winner := a.highestBy
	bid := a.highestBid
	targets := make([]Participant, 0, len(a.participants))
	for _, p := range a.participants {
		targets = append(targets, p)
	}
	a.mu.Unlock()

	for _, t := range targets {
		t.Receive(Message{From: winner, Kind: "sold", Bid: bid})
	}
}

// HighestBid expoe o estado atual (safe).
func (a *AuctionRoom) HighestBid() (string, int64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.highestBy, a.highestBid
}

// Bidder participante concreto que guarda mensagens recebidas.
type Bidder struct {
	id       string
	mu       sync.Mutex
	received []Message
}

func NewBidder(id string) *Bidder       { return &Bidder{id: id} }
func (b *Bidder) ID() string            { return b.id }
func (b *Bidder) Receive(msg Message) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.received = append(b.received, msg)
}
func (b *Bidder) Inbox() []Message {
	b.mu.Lock()
	defer b.mu.Unlock()
	cp := make([]Message, len(b.received))
	copy(cp, b.received)
	return cp
}
