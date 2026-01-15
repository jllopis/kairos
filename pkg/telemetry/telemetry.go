// SPDX-License-Identifier: Apache-2.0
// Package telemetry configures OpenTelemetry exporters and propagators.
// See docs/ERROR_HANDLING.md for error handling integration.
package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/jllopis/kairos/pkg/errors"
)

// ShutdownFunc releases telemetry resources created by Init or InitWithConfig.
type ShutdownFunc func(context.Context) error

// Config controls telemetry exporter behavior and OTLP connection settings.
type Config struct {
	Exporter           string
	OTLPEndpoint       string
	OTLPInsecure       bool
	OTLPTimeoutSeconds int
}

// Init initializes OpenTelemetry with stdout exporters using default settings.
func Init(serviceName, version string) (ShutdownFunc, error) {
	return InitWithConfig(serviceName, version, Config{Exporter: "stdout"})
}

// InitWithConfig initializes OpenTelemetry with the specified exporter config.
func InitWithConfig(serviceName, version string, cfg Config) (ShutdownFunc, error) {
	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(version),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	tp, mp, err := initProviders(res, cfg)
	if err != nil {
		return nil, err
	}

	// Propagators
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Return global shutdown function
	return func(ctx context.Context) error {
		var errs []error
		if err := tp.Shutdown(ctx); err != nil {
			errs = append(errs, err)
		}
		if err := mp.Shutdown(ctx); err != nil {
			errs = append(errs, err)
		}
		if len(errs) > 0 {
			return fmt.Errorf("telemetry shutdown errors: %v", errs)
		}
		return nil
	}, nil
}

func initProviders(res *resource.Resource, cfg Config) (*sdktrace.TracerProvider, *metric.MeterProvider, error) {
	switch cfg.Exporter {
	case "", "stdout":
		return initStdout(res)
	case "none":
		return initNoop(res)
	case "otlp":
		if cfg.OTLPEndpoint == "" {
			return nil, nil, fmt.Errorf("otlp endpoint is required")
		}
		return initOTLP(res, cfg)
	default:
		return nil, nil, fmt.Errorf("unknown telemetry exporter: %s", cfg.Exporter)
	}
}

func initStdout(res *resource.Resource) (*sdktrace.TracerProvider, *metric.MeterProvider, error) {
	traceExporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter, sdktrace.WithBatchTimeout(time.Second)),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	metricExporter, err := stdoutmetric.New()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create metric exporter: %w", err)
	}
	mp := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(metricExporter, metric.WithInterval(time.Minute))),
		metric.WithResource(res),
	)
	otel.SetMeterProvider(mp)
	return tp, mp, nil
}

func initNoop(res *resource.Resource) (*sdktrace.TracerProvider, *metric.MeterProvider, error) {
	tp := sdktrace.NewTracerProvider(sdktrace.WithResource(res))
	otel.SetTracerProvider(tp)
	mp := metric.NewMeterProvider(metric.WithResource(res))
	otel.SetMeterProvider(mp)
	return tp, mp, nil
}

func initOTLP(res *resource.Resource, cfg Config) (*sdktrace.TracerProvider, *metric.MeterProvider, error) {
	timeout := 10 * time.Second
	if cfg.OTLPTimeoutSeconds > 0 {
		timeout = time.Duration(cfg.OTLPTimeoutSeconds) * time.Second
	}
	traceOpts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint),
		otlptracegrpc.WithTimeout(timeout),
	}
	metricOpts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(cfg.OTLPEndpoint),
		otlpmetricgrpc.WithTimeout(timeout),
	}
	if cfg.OTLPInsecure {
		traceOpts = append(traceOpts, otlptracegrpc.WithInsecure())
		metricOpts = append(metricOpts, otlpmetricgrpc.WithInsecure())
	}

	traceExporter, err := otlptracegrpc.New(context.Background(), traceOpts...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create otlp trace exporter: %w", err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter, sdktrace.WithBatchTimeout(time.Second)),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	metricExporter, err := otlpmetricgrpc.New(context.Background(), metricOpts...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create otlp metric exporter: %w", err)
	}
	mp := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(metricExporter, metric.WithInterval(time.Minute))),
		metric.WithResource(res),
	)
	otel.SetMeterProvider(mp)
	return tp, mp, nil
}

// RecordError records a Kairos error with full context to the span.
// This integrates error handling with OTEL observability.
func RecordError(span trace.Span, err error) {
	if err == nil || span == nil {
		return
	}

	// Record the base error
	span.RecordError(err)

	// Extract KairosError for rich context
	if ke, ok := err.(*errors.KairosError); ok {
		// Add error code and recoverable flag as attributes
		span.SetAttributes(
			attribute.String("error.code", string(ke.Code)),
			attribute.Bool("error.recoverable", ke.Recoverable),
		)

		// Add custom attributes from the error
		for k, v := range ke.Attributes {
			span.SetAttributes(attribute.String("error."+k, v))
		}

		// Add context data as span attributes for detailed debugging
		for k, v := range ke.Context {
			span.SetAttributes(attribute.String("error.context."+k, fmt.Sprintf("%v", v)))
		}

		// Log structured error data
		slog.Error("KairosError recorded",
			"code", ke.Code,
			"message", ke.Message,
			"recoverable", ke.Recoverable,
			"context", ke.Context,
			"status_code", ke.StatusCode,
		)
	}
}
