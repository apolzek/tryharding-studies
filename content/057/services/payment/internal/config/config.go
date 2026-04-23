package config

import "os"

type Config struct {
	ServiceName  string
	Port         string
	RedisAddr    string
	RabbitURL    string
	OTLPEndpoint string
}

func Load() Config {
	return Config{
		ServiceName:  env("SERVICE_NAME", "payment"),
		Port:         env("PORT", "8007"),
		RedisAddr:    env("REDIS_ADDR", "redis:6379"),
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
