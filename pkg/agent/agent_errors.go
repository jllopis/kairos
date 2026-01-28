// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

// Package agent implements the LLM-driven agent loop and configuration options.
package agent

import (
	"context"
	"sync"

	"github.com/jllopis/kairos/pkg/core"
	"github.com/jllopis/kairos/pkg/errors"
	"github.com/jllopis/kairos/pkg/telemetry"
)

// ErrorMetricsIntegration provides error metrics integration for agents.
// It wraps the telemetry.ErrorMetrics and provides agent-specific helpers.
type ErrorMetricsIntegration struct {
	metrics *telemetry.ErrorMetrics
	enabled bool
	mu      sync.RWMutex
}

var (
	globalErrorMetrics     *ErrorMetricsIntegration
	globalErrorMetricsOnce sync.Once
)

// InitErrorMetrics initializes the global error metrics for agents.
// This should be called once during application startup.
// Returns nil error metrics if initialization fails (graceful degradation).
func InitErrorMetrics(ctx context.Context) *ErrorMetricsIntegration {
	globalErrorMetricsOnce.Do(func() {
		metrics, err := telemetry.NewErrorMetrics(ctx)
		if err != nil {
			globalErrorMetrics = &ErrorMetricsIntegration{enabled: false}
			return
		}
		globalErrorMetrics = &ErrorMetricsIntegration{
			metrics: metrics,
			enabled: true,
		}
	})
	return globalErrorMetrics
}

// GetErrorMetrics returns the global error metrics integration.
// Returns nil if not initialized.
func GetErrorMetrics() *ErrorMetricsIntegration {
	return globalErrorMetrics
}

// RecordError records an error metric with the appropriate error code and component.
func (e *ErrorMetricsIntegration) RecordError(ctx context.Context, err error, component string) {
	if e == nil || !e.enabled || e.metrics == nil {
		return
	}
	e.metrics.RecordErrorMetric(ctx, err, component)
}

// RecordRecovery records a successful recovery for the given error code.
func (e *ErrorMetricsIntegration) RecordRecovery(ctx context.Context, code errors.ErrorCode) {
	if e == nil || !e.enabled || e.metrics == nil {
		return
	}
	e.metrics.RecordRecovery(ctx, code)
}

// RecordHealthStatus records the health status of a component.
func (e *ErrorMetricsIntegration) RecordHealthStatus(ctx context.Context, component string, status core.HealthStatus) {
	if e == nil || !e.enabled || e.metrics == nil {
		return
	}
	var statusVal int64
	switch status {
	case core.HealthHealthy:
		statusVal = 2
	case core.HealthDegraded:
		statusVal = 1
	default:
		statusVal = 0
	}
	e.metrics.RecordHealthStatus(ctx, component, statusVal)
}

// RecordCircuitBreakerState records the circuit breaker state for a component.
// state: 0=OPEN (failing), 1=HALF_OPEN (testing), 2=CLOSED (healthy)
func (e *ErrorMetricsIntegration) RecordCircuitBreakerState(ctx context.Context, component string, state int64) {
	if e == nil || !e.enabled || e.metrics == nil {
		return
	}
	e.metrics.RecordCircuitBreakerState(ctx, component, state)
}

// WrapLLMError wraps an LLM error with appropriate context.
func WrapLLMError(err error, model string) *errors.KairosError {
	if err == nil {
		return nil
	}
	ke := errors.New(errors.CodeLLMError, "LLM call failed", err).
		WithContext("model", model).
		WithAttribute("llm.model", model).
		WithRecoverable(true)
	return ke
}

// WrapToolError wraps a tool execution error with appropriate context.
func WrapToolError(err error, toolName, toolCallID string) *errors.KairosError {
	if err == nil {
		return nil
	}
	ke := errors.New(errors.CodeToolFailure, "tool execution failed", err).
		WithContext("tool_name", toolName).
		WithContext("tool_call_id", toolCallID).
		WithAttribute("tool.name", toolName).
		WithRecoverable(true)
	return ke
}

// WrapMemoryError wraps a memory system error with appropriate context.
func WrapMemoryError(err error, operation string) *errors.KairosError {
	if err == nil {
		return nil
	}
	ke := errors.New(errors.CodeMemoryError, "memory operation failed", err).
		WithContext("operation", operation).
		WithAttribute("memory.operation", operation).
		WithRecoverable(true)
	return ke
}

// WrapTimeoutError wraps a timeout error with appropriate context.
func WrapTimeoutError(err error, operation string, maxIterations int) *errors.KairosError {
	if err == nil {
		return nil
	}
	ke := errors.New(errors.CodeTimeout, "operation exceeded max iterations", err).
		WithContext("operation", operation).
		WithContext("max_iterations", maxIterations).
		WithRecoverable(false)
	return ke
}

// WrapPlannerError wraps an explicit planner execution error with context.
func WrapPlannerError(err error, planID string) *errors.KairosError {
	if err == nil {
		return nil
	}
	ke := errors.New(errors.CodeInternal, "planner execution failed", err).
		WithContext("plan_id", planID).
		WithAttribute("planner.id", planID).
		WithRecoverable(false)
	return ke
}

// NewInvalidInputError creates a new invalid input error.
func NewInvalidInputError(msg string) *errors.KairosError {
	return errors.New(errors.CodeInvalidInput, msg, nil).
		WithRecoverable(false)
}

// NewNotFoundError creates a new not found error.
func NewNotFoundError(resource, name string) *errors.KairosError {
	return errors.New(errors.CodeNotFound, resource+" not found", nil).
		WithContext("resource", resource).
		WithContext("name", name).
		WithRecoverable(false)
}
