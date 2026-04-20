package main

import (
	"context"
	"sync"
	"sync/atomic"
)

// Event e a unidade publicada no bus.
type Event struct {
	Type    string
	Payload map[string]any
}

// Handler assina um tipo de evento.
type Handler func(ctx context.Context, e Event)

// Subscription representa um registro ativo; chame Unsubscribe para remover.
type Subscription struct {
	id      uint64
	evtType string
	bus     *EventBus
}

// Unsubscribe remove o handler; idempotente.
func (s *Subscription) Unsubscribe() {
	if s == nil || s.bus == nil {
		return
	}
	s.bus.unsubscribe(s.evtType, s.id)
}

// EventBus e um barramento in-memory thread-safe com entrega assincrona.
type EventBus struct {
	mu     sync.RWMutex
	subs   map[string]map[uint64]Handler
	nextID atomic.Uint64
	wg     sync.WaitGroup
}

// NewEventBus cria um bus vazio.
func NewEventBus() *EventBus {
	return &EventBus{subs: map[string]map[uint64]Handler{}}
}

// Subscribe registra um handler para um tipo de evento.
func (b *EventBus) Subscribe(evtType string, h Handler) *Subscription {
	id := b.nextID.Add(1)
	b.mu.Lock()
	if _, ok := b.subs[evtType]; !ok {
		b.subs[evtType] = map[uint64]Handler{}
	}
	b.subs[evtType][id] = h
	b.mu.Unlock()
	return &Subscription{id: id, evtType: evtType, bus: b}
}

func (b *EventBus) unsubscribe(evtType string, id uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if m, ok := b.subs[evtType]; ok {
		delete(m, id)
		if len(m) == 0 {
			delete(b.subs, evtType)
		}
	}
}

// Publish entrega o evento a todos os subscribers de forma assincrona.
// Cada handler roda em goroutine propria; use Wait para aguardar drenagem.
func (b *EventBus) Publish(ctx context.Context, e Event) {
	b.mu.RLock()
	handlers := make([]Handler, 0, len(b.subs[e.Type]))
	for _, h := range b.subs[e.Type] {
		handlers = append(handlers, h)
	}
	b.mu.RUnlock()

	for _, h := range handlers {
		h := h
		b.wg.Add(1)
		go func() {
			defer b.wg.Done()
			if err := ctx.Err(); err != nil {
				return
			}
			h(ctx, e)
		}()
	}
}

// Wait bloqueia ate que todos os handlers em voo terminem.
func (b *EventBus) Wait() { b.wg.Wait() }

// SubscriberCount retorna quantos handlers estao registrados para o tipo.
func (b *EventBus) SubscriberCount(evtType string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subs[evtType])
}
