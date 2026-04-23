package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/tryharding/057/payment/internal/chaos"
	"github.com/tryharding/057/payment/internal/config"
	"github.com/tryharding/057/payment/internal/events"
	"github.com/tryharding/057/payment/internal/handler"
	"github.com/tryharding/057/payment/internal/idempotency"
	"github.com/tryharding/057/payment/internal/payment"
	"github.com/tryharding/057/payment/internal/telemetry"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func main() {
	cfg := config.Load()

	shutdown, err := telemetry.Setup(context.Background(), cfg.ServiceName, cfg.OTLPEndpoint)
	if err != nil {
		log.Fatalf("otel: %v", err)
	}
	defer shutdown(context.Background())

	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	idem := idempotency.New(rdb)

	pub, err := events.NewPublisher(cfg.RabbitURL, "payment.events")
	if err != nil {
		log.Printf("rabbit: %v (continuing without events)", err)
	}

	// Also consume order.events to know when a payment is requested.
	if cons, err := events.NewOrderConsumer(cfg.RabbitURL); err == nil {
		go cons.Run(context.Background(), payment.NewGateway(pub))
	} else {
		log.Printf("consumer: %v (skipping)", err)
	}

	chaosState := chaos.New()
	gw := payment.NewGateway(pub)

	r := gin.New()
	r.Use(gin.Recovery(), otelgin.Middleware(cfg.ServiceName), chaos.Middleware(chaosState))
	handler.Register(r, gw, idem, chaosState)

	go func() {
		if err := r.Run(":" + cfg.Port); err != nil {
			log.Fatalf("run: %v", err)
		}
	}()
	log.Printf("payment listening on :%s", cfg.Port)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
}
