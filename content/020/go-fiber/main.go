package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/contrib/otelfiber"
	"github.com/google/uuid"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	otlpmetricgrpc "go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ── Models ────────────────────────────────────────────────────────────────────

type LoanApplyRequest struct {
	CustomerID string  `json:"customer_id"`
	Amount     float64 `json:"amount"`
	Currency   string  `json:"currency"`
	Purpose    string  `json:"purpose"`
}

type LoanApplyResponse struct {
	ApplicationID string      `json:"application_id"`
	Status        string      `json:"status"`
	Decision      interface{} `json:"decision"`
}

// ── OTel SDK bootstrap ────────────────────────────────────────────────────────
// Only the SDK setup lives here; all span/metric creation is done by
// otelfiber.Middleware() and otelhttp.NewTransport() — no manual instrumentation.

func initOtel(ctx context.Context) (*sdktrace.TracerProvider, *sdkmetric.MeterProvider, error) {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "localhost:4317"
	}

	//nolint:staticcheck
	conn, err := grpc.DialContext(ctx, endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("grpc dial: %w", err)
	}

	serviceName := os.Getenv("OTEL_SERVICE_NAME")
	if serviceName == "" {
		serviceName = "go-fiber"
	}
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion("1.0.0"),
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("resource: %w", err)
	}

	traceExp, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, nil, fmt.Errorf("trace exporter: %w", err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExp),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	metricExp, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, nil, fmt.Errorf("metric exporter: %w", err)
	}
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExp,
			sdkmetric.WithInterval(10*time.Second),
		)),
		sdkmetric.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	return tp, mp, nil
}

// ── Main ──────────────────────────────────────────────────────────────────────

func main() {
	ctx := context.Background()

	tp, mp, err := initOtel(ctx)
	if err != nil {
		log.Fatalf("otel init: %v", err)
	}
	defer func() {
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tp.Shutdown(shutCtx)
		_ = mp.Shutdown(shutCtx)
	}()

	javaSpringURL := os.Getenv("JAVA_SPRING_URL")
	if javaSpringURL == "" {
		javaSpringURL = "http://localhost:8081"
	}

	// otelhttp.NewTransport auto-injects W3C traceparent on every outgoing request
	httpClient := &http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
		Timeout:   30 * time.Second,
	}

	app := fiber.New(fiber.Config{
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	})

	// otelfiber.Middleware() auto-instruments incoming HTTP requests:
	// extracts W3C traceparent, creates server span — no manual code needed.
	app.Use(otelfiber.Middleware())

	// ── POST /api/loan/apply ───────────────────────────────────────────────────
	app.Post("/api/loan/apply", func(c *fiber.Ctx) error {
		var req LoanApplyRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
		}
		if req.CustomerID == "" || req.Amount <= 0 || req.Currency == "" {
			return c.Status(400).JSON(fiber.Map{
				"error": "customer_id, amount (>0), and currency are required",
			})
		}

		applicationID := uuid.New().String()
		// c.UserContext() carries the span context set by otelfiber middleware
		ctx := c.UserContext()

		log.Printf("[go-fiber] application_id=%s customer=%s amount=%.2f %s",
			applicationID, req.CustomerID, req.Amount, req.Currency)

		payload := map[string]interface{}{
			"application_id": applicationID,
			"customer_id":    req.CustomerID,
			"amount":         req.Amount,
			"currency":       req.Currency,
			"purpose":        req.Purpose,
		}
		body, _ := json.Marshal(payload)

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
			javaSpringURL+"/api/loan/enrich", bytes.NewBuffer(body))
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "internal error"})
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("X-Application-ID", applicationID)
		httpReq.Header.Set("X-Source-Service", "go-fiber")

		resp, err := httpClient.Do(httpReq)
		if err != nil {
			return c.Status(502).JSON(fiber.Map{"error": "java-spring unreachable: " + err.Error()})
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)
		var decision interface{}
		_ = json.Unmarshal(respBody, &decision)

		return c.Status(resp.StatusCode).JSON(LoanApplyResponse{
			ApplicationID: applicationID,
			Status:        "processed",
			Decision:      decision,
		})
	})

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "service": "go-fiber"})
	})

	log.Println("[go-fiber] Listening on :8080")
	log.Fatal(app.Listen(":8080"))
}
