package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/tryharding/057/customer/internal/config"
	"github.com/tryharding/057/customer/internal/events"
	"github.com/tryharding/057/customer/internal/handler"
	"github.com/tryharding/057/customer/internal/repo"
	"github.com/tryharding/057/customer/internal/service"
	"github.com/tryharding/057/customer/internal/telemetry"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func main() {
	cfg := config.Load()
	shutdown, err := telemetry.Setup(context.Background(), cfg.ServiceName, cfg.OTLPEndpoint)
	if err != nil {
		log.Fatalf("otel: %v", err)
	}
	defer shutdown(context.Background())

	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()
	if err := repo.Migrate(context.Background(), pool); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	pub, err := events.NewPublisher(cfg.RabbitURL, "customer.events")
	if err != nil {
		log.Printf("rabbit: %v (continuing without events)", err)
	}

	custRepo := repo.NewCustomerRepo(pool)
	svc := service.NewCustomerService(custRepo, pub)

	r := gin.New()
	r.Use(gin.Recovery(), otelgin.Middleware(cfg.ServiceName))
	handler.Register(r, svc)

	go func() {
		if err := r.Run(":" + cfg.Port); err != nil {
			log.Fatalf("run: %v", err)
		}
	}()
	log.Printf("customer listening on :%s", cfg.Port)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
}
