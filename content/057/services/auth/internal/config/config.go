package config

import "os"

type Config struct {
	ServiceName  string
	Port         string
	DatabaseURL  string
	RedisAddr    string
	JWTSecret    string
	OTLPEndpoint string
}

func Load() Config {
	return Config{
		ServiceName:  env("SERVICE_NAME", "auth"),
		Port:         env("PORT", "8001"),
		DatabaseURL:  env("DATABASE_URL", "postgres://postgres:postgres@postgres:5432/auth?sslmode=disable"),
		RedisAddr:    env("REDIS_ADDR", "redis:6379"),
		JWTSecret:    env("JWT_SECRET", "dev-secret-change-me"),
		OTLPEndpoint: env("OTEL_EXPORTER_OTLP_ENDPOINT", "otel-collector:4317"),
	}
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
