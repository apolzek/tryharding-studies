package events

import (
	"context"
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Processor is anything that can charge a given order.
type Processor interface {
	Charge(ctx context.Context, orderID string, amount float64, idempotencyKey string) (status int, resp any, err error)
}

type OrderConsumer struct {
	conn *amqp.Connection
}

func NewOrderConsumer(url string) (*OrderConsumer, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}
	return &OrderConsumer{conn: conn}, nil
}

func (c *OrderConsumer) Run(ctx context.Context, p Processor) {
	ch, err := c.conn.Channel()
	if err != nil {
		log.Printf("channel: %v", err)
		return
	}
	ex := "order.events"
	if err := ch.ExchangeDeclare(ex, "topic", true, false, false, false, nil); err != nil {
		log.Printf("exchange: %v", err)
		return
	}
	q, err := ch.QueueDeclare("payment.order-events", true, false, false, false, nil)
	if err != nil {
		log.Printf("queue: %v", err)
		return
	}
	if err := ch.QueueBind(q.Name, "order.payment_requested", ex, false, nil); err != nil {
		log.Printf("bind: %v", err)
		return
	}
	msgs, err := ch.Consume(q.Name, "", false, false, false, false, nil)
	if err != nil {
		log.Printf("consume: %v", err)
		return
	}

	log.Printf("payment ← order.payment_requested consumer ready")

	for msg := range msgs {
		var payload struct {
			OrderID string  `json:"order_id"`
			Amount  float64 `json:"amount"`
		}
		if err := json.Unmarshal(msg.Body, &payload); err != nil {
			log.Printf("bad payload: %v", err)
			_ = msg.Nack(false, false)
			continue
		}
		_, _, err := p.Charge(ctx, payload.OrderID, payload.Amount, "auto-"+payload.OrderID)
		if err != nil {
			log.Printf("charge error (requeueing): %v", err)
			_ = msg.Nack(false, true)
			continue
		}
		_ = msg.Ack(false)
	}
}
