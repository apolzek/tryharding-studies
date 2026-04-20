package main

import (
	"context"
	"fmt"
	"sync/atomic"
)

func main() {
	ctx := context.Background()
	bus := NewEventBus()

	var emails, stock, metrics atomic.Int64

	bus.Subscribe("OrderPlaced", func(ctx context.Context, e Event) {
		emails.Add(1)
		fmt.Printf("[email] pedido %v\n", e.Payload["order_id"])
	})
	bus.Subscribe("OrderPlaced", func(ctx context.Context, e Event) {
		stock.Add(1)
		fmt.Printf("[stock] reservando itens do pedido %v\n", e.Payload["order_id"])
	})
	sMetrics := bus.Subscribe("OrderPlaced", func(ctx context.Context, e Event) {
		metrics.Add(1)
		fmt.Printf("[metrics] contador++ pedido %v\n", e.Payload["order_id"])
	})

	bus.Publish(ctx, Event{Type: "OrderPlaced", Payload: map[string]any{"order_id": "A-1"}})
	bus.Publish(ctx, Event{Type: "OrderPlaced", Payload: map[string]any{"order_id": "A-2"}})

	bus.Wait()

	// Demonstra unsubscribe.
	sMetrics.Unsubscribe()
	bus.Publish(ctx, Event{Type: "OrderPlaced", Payload: map[string]any{"order_id": "A-3"}})
	bus.Wait()

	fmt.Printf("totais: emails=%d stock=%d metrics=%d\n",
		emails.Load(), stock.Load(), metrics.Load())
}
