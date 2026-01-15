// SPDX-License-Identifier: Apache-2.0
// Package telemetry provides observability for Kairos error handling.
// See docs/ERROR_HANDLING.md for metric integration patterns.
package telemetry

import (
	"context"
	"testing"

	"github.com/jllopis/kairos/pkg/errors"
)

func TestNewErrorMetrics(t *testing.T) {
	em, err := NewErrorMetrics(context.Background())
	if err != nil {
		t.Fatalf("failed to create error metrics: %v", err)
	}
	if em == nil {
		t.Fatal("expected non-nil ErrorMetrics")
	}
}

func TestRecordErrorMetric(t *testing.T) {
	em, _ := NewErrorMetrics(context.Background())
	ctx := context.Background()

	// Record a KairosError
	ke := errors.New(errors.CodeToolFailure, "tool failed", nil)
	em.RecordErrorMetric(ctx, ke, "llm-service")

	// Record a generic error
	em.RecordErrorMetric(ctx, errors.New(errors.CodeInternal, "generic error", nil), "worker")

	// Should not panic with nil error or metrics
	em.RecordErrorMetric(ctx, nil, "service")
	em.RecordErrorMetric(ctx, ke, "")

	// Nil metrics should not panic
	var nilMetrics *ErrorMetrics
	nilMetrics.RecordErrorMetric(ctx, ke, "service")
}

func TestRecordRecovery(t *testing.T) {
	em, _ := NewErrorMetrics(context.Background())
	ctx := context.Background()

	em.RecordRecovery(ctx, errors.CodeToolFailure)
	em.RecordRecovery(ctx, errors.CodeTimeout)
	em.RecordRecovery(ctx, errors.CodeRateLimit)

	var nilMetrics *ErrorMetrics
	nilMetrics.RecordRecovery(ctx, errors.CodeToolFailure)
}

func TestRecordErrorRate(t *testing.T) {
	em, _ := NewErrorMetrics(context.Background())
	ctx := context.Background()

	em.RecordErrorRate(ctx, "llm-service", 2.5)
	em.RecordErrorRate(ctx, "agent-pool", 0.1)
	em.RecordErrorRate(ctx, "memory", 0.0)

	var nilMetrics *ErrorMetrics
	nilMetrics.RecordErrorRate(ctx, "service", 1.5)
}

func TestRecordHealthStatus(t *testing.T) {
	em, _ := NewErrorMetrics(context.Background())
	ctx := context.Background()

	// 0 = unhealthy, 1 = degraded, 2 = healthy
	em.RecordHealthStatus(ctx, "llm-service", 2)
	em.RecordHealthStatus(ctx, "cache", 1)
	em.RecordHealthStatus(ctx, "database", 0)

	var nilMetrics *ErrorMetrics
	nilMetrics.RecordHealthStatus(ctx, "service", 2)
}

func TestRecordCircuitBreakerState(t *testing.T) {
	em, _ := NewErrorMetrics(context.Background())
	ctx := context.Background()

	// 0 = open, 1 = half-open, 2 = closed
	em.RecordCircuitBreakerState(ctx, "api-client", 2)
	em.RecordCircuitBreakerState(ctx, "external-service", 1)
	em.RecordCircuitBreakerState(ctx, "failing-service", 0)

	var nilMetrics *ErrorMetrics
	nilMetrics.RecordCircuitBreakerState(ctx, "service", 2)
}

func TestConcurrentMetrics(t *testing.T) {
	em, _ := NewErrorMetrics(context.Background())
	ctx := context.Background()

	// Simulate concurrent recording
	done := make(chan bool, 3)

	go func() {
		ke := errors.New(errors.CodeLLMError, "model overloaded", nil)
		for i := 0; i < 10; i++ {
			em.RecordErrorMetric(ctx, ke, "llm-1")
			em.RecordRecovery(ctx, errors.CodeLLMError)
		}
		done <- true
	}()

	go func() {
		ke := errors.New(errors.CodeToolFailure, "tool timeout", nil)
		for i := 0; i < 10; i++ {
			em.RecordErrorMetric(ctx, ke, "tool-executor")
			em.RecordErrorRate(ctx, "tool-executor", 1.5+float64(i)*0.1)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 10; i++ {
			em.RecordHealthStatus(ctx, "service", int64(i%3))
			em.RecordCircuitBreakerState(ctx, "endpoint", int64(i%3))
		}
		done <- true
	}()

	<-done
	<-done
	<-done
}
