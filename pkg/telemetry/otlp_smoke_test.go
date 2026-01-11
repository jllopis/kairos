package telemetry

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func TestOTLPSmoke(t *testing.T) {
	if os.Getenv("KAIROS_OTLP_SMOKE_TEST") != "1" {
		t.Skip("set KAIROS_OTLP_SMOKE_TEST=1 to run")
	}

	endpoint := os.Getenv("KAIROS_TELEMETRY_OTLP_ENDPOINT")
	if endpoint == "" {
		t.Skip("set KAIROS_TELEMETRY_OTLP_ENDPOINT for OTLP smoke test")
	}

	cfg := Config{
		Exporter:     "otlp",
		OTLPEndpoint: endpoint,
	}
	if os.Getenv("KAIROS_TELEMETRY_OTLP_INSECURE") == "true" {
		cfg.OTLPInsecure = true
	}
	if raw := os.Getenv("KAIROS_TELEMETRY_OTLP_TIMEOUT_SECONDS"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			cfg.OTLPTimeoutSeconds = parsed
		}
	}

	shutdown, err := InitWithConfig("telemetry-smoke-test", "v0.2.5", cfg)
	if err != nil {
		t.Fatalf("failed to init telemetry: %v", err)
	}

	tracer := otel.Tracer("kairos/telemetry-smoke")
	ctx, span := tracer.Start(context.Background(), "smoke.span")
	span.SetAttributes(attribute.String("smoke.test", "otlp"))
	span.End()

	meter := otel.Meter("kairos/telemetry-smoke")
	counter, err := meter.Int64Counter("kairos.telemetry.smoke.counter")
	if err == nil {
		counter.Add(ctx, 1, metric.WithAttributes(attribute.String("smoke.test", "otlp")))
	}

	time.Sleep(2 * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := shutdown(ctx); err != nil {
		t.Fatalf("telemetry shutdown failed: %v", err)
	}
}
