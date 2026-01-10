package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// ShutdownFunc is a function that cleans up telemetry resources.
type ShutdownFunc func(context.Context) error

// Config controls telemetry exporter behavior.
type Config struct {
	Exporter     string
	OTLPEndpoint string
	OTLPInsecure bool
}

// Init initializes the OpenTelemetry SDK with stdout exporters.
func Init(serviceName, version string) (ShutdownFunc, error) {
	return InitWithConfig(serviceName, version, Config{Exporter: "stdout"})
}

// InitWithConfig initializes the OpenTelemetry SDK with the specified exporter.
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

func initProviders(res *resource.Resource, cfg Config) (*trace.TracerProvider, *metric.MeterProvider, error) {
	switch cfg.Exporter {
	case "", "stdout":
		return initStdout(res)
	case "otlp":
		if cfg.OTLPEndpoint == "" {
			return nil, nil, fmt.Errorf("otlp endpoint is required")
		}
		return initOTLP(res, cfg)
	default:
		return nil, nil, fmt.Errorf("unknown telemetry exporter: %s", cfg.Exporter)
	}
}

func initStdout(res *resource.Resource) (*trace.TracerProvider, *metric.MeterProvider, error) {
	traceExporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}
	tp := trace.NewTracerProvider(
		trace.WithBatcher(traceExporter, trace.WithBatchTimeout(time.Second)),
		trace.WithResource(res),
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

func initOTLP(res *resource.Resource, cfg Config) (*trace.TracerProvider, *metric.MeterProvider, error) {
	traceOpts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint),
	}
	metricOpts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(cfg.OTLPEndpoint),
	}
	if cfg.OTLPInsecure {
		traceOpts = append(traceOpts, otlptracegrpc.WithInsecure())
		metricOpts = append(metricOpts, otlpmetricgrpc.WithInsecure())
	}

	traceExporter, err := otlptracegrpc.New(context.Background(), traceOpts...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create otlp trace exporter: %w", err)
	}
	tp := trace.NewTracerProvider(
		trace.WithBatcher(traceExporter, trace.WithBatchTimeout(time.Second)),
		trace.WithResource(res),
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
