package main

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
)

func TestPublishEntregaParaTodos(t *testing.T) {
	tests := []struct {
		name      string
		subsCount int
		publishes int
	}{
		{"1 sub 1 pub", 1, 1},
		{"3 subs 5 pubs", 3, 5},
		{"10 subs 2 pubs", 10, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bus := NewEventBus()
			var count atomic.Int64

			for i := 0; i < tt.subsCount; i++ {
				bus.Subscribe("evt", func(ctx context.Context, e Event) {
					count.Add(1)
				})
			}

			ctx := context.Background()
			for i := 0; i < tt.publishes; i++ {
				bus.Publish(ctx, Event{Type: "evt"})
			}
			bus.Wait()

			want := int64(tt.subsCount * tt.publishes)
			if count.Load() != want {
				t.Fatalf("esperava %d, got %d", want, count.Load())
			}
		})
	}
}

func TestUnsubscribe(t *testing.T) {
	bus := NewEventBus()
	var n atomic.Int64
	sub := bus.Subscribe("x", func(ctx context.Context, e Event) {
		n.Add(1)
	})

	bus.Publish(context.Background(), Event{Type: "x"})
	bus.Wait()

	sub.Unsubscribe()
	bus.Publish(context.Background(), Event{Type: "x"})
	bus.Wait()

	if n.Load() != 1 {
		t.Fatalf("esperava 1, got %d", n.Load())
	}
	if bus.SubscriberCount("x") != 0 {
		t.Fatalf("subscribers nao zeraram")
	}

	// Unsubscribe idempotente.
	sub.Unsubscribe()
}

func TestConcurrentPublishSubscribe(t *testing.T) {
	bus := NewEventBus()
	var n atomic.Int64

	const workers = 32
	const msgs = 100

	var subs []*Subscription
	var subMu sync.Mutex
	for i := 0; i < workers; i++ {
		s := bus.Subscribe("c", func(ctx context.Context, e Event) {
			n.Add(1)
		})
		subs = append(subs, s)
	}

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < msgs; i++ {
				bus.Publish(context.Background(), Event{Type: "c"})
			}
		}()
	}

	// Subscribe/unsubscribe concorrente para exercitar locks.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			s := bus.Subscribe("c", func(ctx context.Context, e Event) {})
			subMu.Lock()
			subs = append(subs, s)
			subMu.Unlock()
			s.Unsubscribe()
		}
	}()

	wg.Wait()
	bus.Wait()

	if n.Load() < int64(workers*msgs) {
		t.Fatalf("entrega incompleta: got %d", n.Load())
	}
}

func TestPublishCtxCancelado(t *testing.T) {
	bus := NewEventBus()
	var n atomic.Int64
	bus.Subscribe("y", func(ctx context.Context, e Event) {
		n.Add(1)
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	bus.Publish(ctx, Event{Type: "y"})
	bus.Wait()

	if n.Load() != 0 {
		t.Fatalf("handler nao deveria executar com ctx cancelado, got %d", n.Load())
	}
}

func TestSubscriberCount(t *testing.T) {
	bus := NewEventBus()
	if bus.SubscriberCount("z") != 0 {
		t.Fatalf("esperava 0")
	}
	s1 := bus.Subscribe("z", func(ctx context.Context, e Event) {})
	s2 := bus.Subscribe("z", func(ctx context.Context, e Event) {})
	if bus.SubscriberCount("z") != 2 {
		t.Fatalf("esperava 2")
	}
	s1.Unsubscribe()
	if bus.SubscriberCount("z") != 1 {
		t.Fatalf("esperava 1")
	}
	s2.Unsubscribe()
	if bus.SubscriberCount("z") != 0 {
		t.Fatalf("esperava 0 apos todos unsubscribe")
	}
}

func TestSubscriptionNilSafe(t *testing.T) {
	var s *Subscription
	s.Unsubscribe() // nao deve panic
}

func TestPublishSemSubscribers(t *testing.T) {
	bus := NewEventBus()
	bus.Publish(context.Background(), Event{Type: "vazio"})
	bus.Wait()
}

func TestMainDemo(t *testing.T) {
	main()
}
