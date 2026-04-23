package config

import "os"

type Config struct {
	ServiceName  string
	Port         string
	DatabaseURL  string
	RabbitURL    string
	OTLPEndpoint string
}

func Load() Config {
	return Config{
		ServiceName:  env("SERVICE_NAME", "customer"),
		Port:         env("PORT", "8002"),
		DatabaseURL:  env("DATABASE_URL", "postgres://postgres:postgres@postgres:5432/customer?sslmode=disable"),
		RabbitURL:    env("RABBIT_URL", "amqp://guest:guest@rabbitmq:5672/"),
		OTLPEndpoint: env("OTEL_EXPORTER_OTLP_ENDPOINT", "otel-collector:4317"),
	}
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
