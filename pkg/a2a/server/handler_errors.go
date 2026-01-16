// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

// Package server implements the A2A gRPC server binding and core handlers.
package server

import (
	"context"
	"sync"

	"github.com/jllopis/kairos/pkg/core"
	"github.com/jllopis/kairos/pkg/errors"
	"github.com/jllopis/kairos/pkg/telemetry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrorMetricsIntegration provides error metrics integration for A2A server.
type ErrorMetricsIntegration struct {
	metrics *telemetry.ErrorMetrics
	enabled bool
	mu      sync.RWMutex
}

var (
	serverErrorMetrics     *ErrorMetricsIntegration
	serverErrorMetricsOnce sync.Once
)

// InitServerErrorMetrics initializes the global error metrics for A2A server.
func InitServerErrorMetrics(ctx context.Context) *ErrorMetricsIntegration {
	serverErrorMetricsOnce.Do(func() {
		metrics, err := telemetry.NewErrorMetrics(ctx)
		if err != nil {
			serverErrorMetrics = &ErrorMetricsIntegration{enabled: false}
			return
		}
		serverErrorMetrics = &ErrorMetricsIntegration{
			metrics: metrics,
			enabled: true,
		}
	})
	return serverErrorMetrics
}

// GetServerErrorMetrics returns the global server error metrics integration.
func GetServerErrorMetrics() *ErrorMetricsIntegration {
	return serverErrorMetrics
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

// ToGRPCStatus converts a KairosError to a gRPC status error.
// It maps error codes to appropriate gRPC codes and preserves error context.
func ToGRPCStatus(err error) error {
	if err == nil {
		return nil
	}

	ke := errors.AsKairosError(err)
	if ke == nil {
		return status.Error(codes.Internal, err.Error())
	}

	grpcCode := mapErrorCodeToGRPC(ke.Code)
	return status.Error(grpcCode, ke.Message)
}

// ToGRPCStatusWithDetails converts a KairosError to a gRPC status with details.
func ToGRPCStatusWithDetails(err error) error {
	if err == nil {
		return nil
	}

	ke := errors.AsKairosError(err)
	if ke == nil {
		return status.Error(codes.Internal, err.Error())
	}

	grpcCode := mapErrorCodeToGRPC(ke.Code)
	st := status.New(grpcCode, ke.Message)

	// Add details if available (would require proto definition for error details)
	// For now, just return the status
	return st.Err()
}

// mapErrorCodeToGRPC maps KairosError codes to gRPC codes.
func mapErrorCodeToGRPC(code errors.ErrorCode) codes.Code {
	switch code {
	case errors.CodeInvalidInput:
		return codes.InvalidArgument
	case errors.CodeNotFound:
		return codes.NotFound
	case errors.CodeUnauthorized:
		return codes.Unauthenticated
	case errors.CodeTimeout:
		return codes.DeadlineExceeded
	case errors.CodeRateLimit:
		return codes.ResourceExhausted
	case errors.CodeToolFailure:
		return codes.FailedPrecondition
	case errors.CodeLLMError:
		return codes.Unavailable
	case errors.CodeMemoryError:
		return codes.DataLoss
	case errors.CodeContextLost:
		return codes.Canceled
	case errors.CodeInternal:
		return codes.Internal
	default:
		return codes.Unknown
	}
}

// WrapTaskError wraps an error that occurred during task execution.
func WrapTaskError(err error, taskID string) *errors.KairosError {
	if err == nil {
		return nil
	}
	return errors.New(errors.CodeInternal, "task execution failed", err).
		WithContext("task_id", taskID).
		WithRecoverable(false)
}

// WrapStoreError wraps an error that occurred during store operations.
func WrapStoreError(err error, operation, taskID string) *errors.KairosError {
	if err == nil {
		return nil
	}
	return errors.New(errors.CodeInternal, "store operation failed", err).
		WithContext("operation", operation).
		WithContext("task_id", taskID).
		WithRecoverable(true)
}

// NewTaskNotFoundError creates a task not found error.
func NewTaskNotFoundError(taskID string) *errors.KairosError {
	return errors.New(errors.CodeNotFound, "task not found", nil).
		WithContext("task_id", taskID).
		WithRecoverable(false)
}

// NewInvalidRequestError creates an invalid request error.
func NewInvalidRequestError(msg string) *errors.KairosError {
	return errors.New(errors.CodeInvalidInput, msg, nil).
		WithRecoverable(false)
}

// NewConfigurationError creates a configuration error.
func NewConfigurationError(component string) *errors.KairosError {
	return errors.New(errors.CodeInternal, component+" not configured", nil).
		WithContext("component", component).
		WithRecoverable(false)
}

// NewPolicyDeniedError creates a policy denied error.
func NewPolicyDeniedError(reason string) *errors.KairosError {
	return errors.New(errors.CodeUnauthorized, "policy denied", nil).
		WithContext("reason", reason).
		WithRecoverable(false)
}
