// SPDX-License-Identifier: Apache-2.0
// Package telemetry provides observability for Kairos error handling.
// See docs/ERROR_HANDLING.md for metric integration patterns.
package telemetry

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/jllopis/kairos/pkg/errors"
)

// ErrorMetrics tracks error rates, types, and recovery patterns for production monitoring.
type ErrorMetrics struct {
	// errorCounter tracks total errors by code and component
	errorCounter metric.Int64Counter

	// recoveryCounter tracks successful recoveries
	recoveryCounter metric.Int64Counter

	// errorRateGauge tracks error rate (errors per minute)
	errorRateGauge metric.Float64Gauge

	// healthStatusGauge tracks component health (0=unhealthy, 1=degraded, 2=healthy)
	healthStatusGauge metric.Int64Gauge

	// circuitBreakerStateGauge tracks circuit breaker state per component
	circuitBreakerStateGauge metric.Int64Gauge

	mu sync.RWMutex
}

// NewErrorMetrics creates a new error metrics tracker with OTEL meters.
func NewErrorMetrics(ctx context.Context) (*ErrorMetrics, error) {
	meter := otel.Meter("kairos/errors")

	errorCounter, err := meter.Int64Counter(
		"kairos.errors.total",
		metric.WithDescription("Total errors by code and component"),
	)
	if err != nil {
		return nil, err
	}

	recoveryCounter, err := meter.Int64Counter(
		"kairos.errors.recovered",
		metric.WithDescription("Successful error recoveries by code"),
	)
	if err != nil {
		return nil, err
	}

	errorRateGauge, err := meter.Float64Gauge(
		"kairos.errors.rate",
		metric.WithDescription("Error rate per minute by component"),
	)
	if err != nil {
		return nil, err
	}

	healthStatusGauge, err := meter.Int64Gauge(
		"kairos.health.status",
		metric.WithDescription("Component health status (0=unhealthy, 1=degraded, 2=healthy)"),
	)
	if err != nil {
		return nil, err
	}

	circuitBreakerStateGauge, err := meter.Int64Gauge(
		"kairos.circuitbreaker.state",
		metric.WithDescription("Circuit breaker state per component (0=open, 1=half-open, 2=closed)"),
	)
	if err != nil {
		return nil, err
	}

	return &ErrorMetrics{
		errorCounter:             errorCounter,
		recoveryCounter:          recoveryCounter,
		errorRateGauge:           errorRateGauge,
		healthStatusGauge:        healthStatusGauge,
		circuitBreakerStateGauge: circuitBreakerStateGauge,
	}, nil
}

// RecordErrorMetric increments the error counter for the given error code and component.
// This is called by error handling code to track error rates.
func (em *ErrorMetrics) RecordErrorMetric(ctx context.Context, err error, component string) {
	if em == nil || err == nil {
		return
	}

	em.mu.RLock()
	defer em.mu.RUnlock()

	if ke, ok := err.(*errors.KairosError); ok {
		em.errorCounter.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("error.code", string(ke.Code)),
				attribute.String("component", component),
				attribute.String("recoverable", ke.RecoverableString()),
			),
		)
	} else {
		// Generic error
		em.errorCounter.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("error.code", "UNKNOWN"),
				attribute.String("component", component),
				attribute.String("recoverable", "unknown"),
			),
		)
	}
}

// RecordRecovery increments the recovery counter for the given error code.
// This is called when an error is successfully handled (retry succeeded, fallback used, etc).
func (em *ErrorMetrics) RecordRecovery(ctx context.Context, errorCode errors.ErrorCode) {
	if em == nil {
		return
	}

	em.mu.RLock()
	defer em.mu.RUnlock()

	em.recoveryCounter.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("error.code", string(errorCode)),
		),
	)
}

// RecordErrorRate records the current error rate for a component (errors per minute).
func (em *ErrorMetrics) RecordErrorRate(ctx context.Context, component string, ratePerMinute float64) {
	if em == nil {
		return
	}

	em.mu.RLock()
	defer em.mu.RUnlock()

	em.errorRateGauge.Record(ctx, ratePerMinute,
		metric.WithAttributes(
			attribute.String("component", component),
		),
	)
}

// RecordHealthStatus records the health status of a component (0=unhealthy, 1=degraded, 2=healthy).
func (em *ErrorMetrics) RecordHealthStatus(ctx context.Context, component string, status int64) {
	if em == nil {
		return
	}

	em.mu.RLock()
	defer em.mu.RUnlock()

	em.healthStatusGauge.Record(ctx, status,
		metric.WithAttributes(
			attribute.String("component", component),
		),
	)
}

// RecordCircuitBreakerState records the circuit breaker state (0=open, 1=half-open, 2=closed).
func (em *ErrorMetrics) RecordCircuitBreakerState(ctx context.Context, component string, state int64) {
	if em == nil {
		return
	}

	em.mu.RLock()
	defer em.mu.RUnlock()

	em.circuitBreakerStateGauge.Record(ctx, state,
		metric.WithAttributes(
			attribute.String("component", component),
		),
	)
}
