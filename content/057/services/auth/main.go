package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tryharding/057/auth/internal/config"
	"github.com/tryharding/057/auth/internal/handler"
	"github.com/tryharding/057/auth/internal/repo"
	"github.com/tryharding/057/auth/internal/service"
	"github.com/tryharding/057/auth/internal/telemetry"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func main() {
	cfg := config.Load()

	shutdown, err := telemetry.Setup(context.Background(), cfg.ServiceName, cfg.OTLPEndpoint)
	if err != nil {
		log.Fatalf("telemetry setup: %v", err)
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

	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})

	userRepo := repo.NewUserRepo(pool)
	authSvc := service.NewAuthService(userRepo, rdb, cfg.JWTSecret)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(otelgin.Middleware(cfg.ServiceName))
	handler.Register(r, authSvc)

	go func() {
		if err := r.Run(":" + cfg.Port); err != nil {
			log.Fatalf("run: %v", err)
		}
	}()
	log.Printf("auth listening on :%s", cfg.Port)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	time.Sleep(200 * time.Millisecond)
}
