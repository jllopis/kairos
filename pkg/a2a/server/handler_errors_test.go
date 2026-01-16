// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"testing"

	"github.com/jllopis/kairos/pkg/core"
	"github.com/jllopis/kairos/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestToGRPCStatus_NilError(t *testing.T) {
	result := ToGRPCStatus(nil)
	if result != nil {
		t.Errorf("ToGRPCStatus(nil) = %v, want nil", result)
	}
}

func TestToGRPCStatus_KairosError(t *testing.T) {
	tests := []struct {
		name     string
		code     errors.ErrorCode
		wantCode codes.Code
	}{
		{"InvalidInput", errors.CodeInvalidInput, codes.InvalidArgument},
		{"NotFound", errors.CodeNotFound, codes.NotFound},
		{"Unauthorized", errors.CodeUnauthorized, codes.Unauthenticated},
		{"Timeout", errors.CodeTimeout, codes.DeadlineExceeded},
		{"RateLimit", errors.CodeRateLimit, codes.ResourceExhausted},
		{"ToolFailure", errors.CodeToolFailure, codes.FailedPrecondition},
		{"LLMError", errors.CodeLLMError, codes.Unavailable},
		{"MemoryError", errors.CodeMemoryError, codes.DataLoss},
		{"ContextLost", errors.CodeContextLost, codes.Canceled},
		{"Internal", errors.CodeInternal, codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ke := errors.New(tt.code, "test message", nil)
			result := ToGRPCStatus(ke)
			if result == nil {
				t.Fatal("ToGRPCStatus() = nil, want non-nil")
			}

			st, ok := status.FromError(result)
			if !ok {
				t.Fatal("ToGRPCStatus() did not return a gRPC status error")
			}
			if st.Code() != tt.wantCode {
				t.Errorf("ToGRPCStatus().Code() = %v, want %v", st.Code(), tt.wantCode)
			}
		})
	}
}

func TestToGRPCStatus_StandardError(t *testing.T) {
	err := errors.New(errors.CodeInternal, "test error", nil)
	result := ToGRPCStatus(err)

	st, ok := status.FromError(result)
	if !ok {
		t.Fatal("ToGRPCStatus() did not return a gRPC status error")
	}
	if st.Code() != codes.Internal {
		t.Errorf("ToGRPCStatus().Code() = %v, want %v", st.Code(), codes.Internal)
	}
}

func TestToGRPCStatusWithDetails_NilError(t *testing.T) {
	result := ToGRPCStatusWithDetails(nil)
	if result != nil {
		t.Errorf("ToGRPCStatusWithDetails(nil) = %v, want nil", result)
	}
}

func TestWrapTaskError(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		taskID  string
		wantNil bool
	}{
		{"nil error", nil, "task-123", true},
		{"with error", errors.New(errors.CodeInternal, "test", nil), "task-123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ke := WrapTaskError(tt.err, tt.taskID)
			if tt.wantNil {
				if ke != nil {
					t.Errorf("WrapTaskError() = %v, want nil", ke)
				}
				return
			}
			if ke == nil {
				t.Fatal("WrapTaskError() = nil, want non-nil")
			}
			if ke.Code != errors.CodeInternal {
				t.Errorf("WrapTaskError().Code = %v, want %v", ke.Code, errors.CodeInternal)
			}
			if ke.Context["task_id"] != tt.taskID {
				t.Errorf("WrapTaskError().Context[task_id] = %v, want %v", ke.Context["task_id"], tt.taskID)
			}
		})
	}
}

func TestWrapStoreError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		operation string
		taskID    string
		wantNil   bool
	}{
		{"nil error", nil, "create", "task-123", true},
		{"with error", errors.New(errors.CodeInternal, "test", nil), "update", "task-456", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ke := WrapStoreError(tt.err, tt.operation, tt.taskID)
			if tt.wantNil {
				if ke != nil {
					t.Errorf("WrapStoreError() = %v, want nil", ke)
				}
				return
			}
			if ke == nil {
				t.Fatal("WrapStoreError() = nil, want non-nil")
			}
			if ke.Context["operation"] != tt.operation {
				t.Errorf("WrapStoreError().Context[operation] = %v, want %v", ke.Context["operation"], tt.operation)
			}
			if ke.Context["task_id"] != tt.taskID {
				t.Errorf("WrapStoreError().Context[task_id] = %v, want %v", ke.Context["task_id"], tt.taskID)
			}
			if !ke.Recoverable {
				t.Error("WrapStoreError().Recoverable = false, want true")
			}
		})
	}
}

func TestNewTaskNotFoundError(t *testing.T) {
	ke := NewTaskNotFoundError("task-123")
	if ke == nil {
		t.Fatal("NewTaskNotFoundError() = nil, want non-nil")
	}
	if ke.Code != errors.CodeNotFound {
		t.Errorf("NewTaskNotFoundError().Code = %v, want %v", ke.Code, errors.CodeNotFound)
	}
	if ke.Context["task_id"] != "task-123" {
		t.Errorf("NewTaskNotFoundError().Context[task_id] = %v, want task-123", ke.Context["task_id"])
	}
	if ke.Recoverable {
		t.Error("NewTaskNotFoundError().Recoverable = true, want false")
	}
}

func TestNewInvalidRequestError(t *testing.T) {
	ke := NewInvalidRequestError("test message")
	if ke == nil {
		t.Fatal("NewInvalidRequestError() = nil, want non-nil")
	}
	if ke.Code != errors.CodeInvalidInput {
		t.Errorf("NewInvalidRequestError().Code = %v, want %v", ke.Code, errors.CodeInvalidInput)
	}
	if ke.Message != "test message" {
		t.Errorf("NewInvalidRequestError().Message = %v, want 'test message'", ke.Message)
	}
}

func TestNewConfigurationError(t *testing.T) {
	ke := NewConfigurationError("handler")
	if ke == nil {
		t.Fatal("NewConfigurationError() = nil, want non-nil")
	}
	if ke.Code != errors.CodeInternal {
		t.Errorf("NewConfigurationError().Code = %v, want %v", ke.Code, errors.CodeInternal)
	}
	if ke.Context["component"] != "handler" {
		t.Errorf("NewConfigurationError().Context[component] = %v, want handler", ke.Context["component"])
	}
}

func TestNewPolicyDeniedError(t *testing.T) {
	ke := NewPolicyDeniedError("access denied")
	if ke == nil {
		t.Fatal("NewPolicyDeniedError() = nil, want non-nil")
	}
	if ke.Code != errors.CodeUnauthorized {
		t.Errorf("NewPolicyDeniedError().Code = %v, want %v", ke.Code, errors.CodeUnauthorized)
	}
	if ke.Context["reason"] != "access denied" {
		t.Errorf("NewPolicyDeniedError().Context[reason] = %v, want 'access denied'", ke.Context["reason"])
	}
}

func TestServerErrorMetricsIntegration(t *testing.T) {
	ctx := context.Background()

	// Initialize server error metrics
	em := InitServerErrorMetrics(ctx)
	if em == nil {
		t.Fatal("InitServerErrorMetrics() = nil, want non-nil")
	}

	// Get global metrics
	globalEM := GetServerErrorMetrics()
	if globalEM == nil {
		t.Fatal("GetServerErrorMetrics() = nil, want non-nil")
	}

	// Test RecordError (should not panic)
	testErr := errors.New(errors.CodeToolFailure, "test", nil)
	em.RecordError(ctx, testErr, "test-server-component")

	// Test RecordRecovery
	em.RecordRecovery(ctx, errors.CodeToolFailure)

	// Test RecordHealthStatus
	em.RecordHealthStatus(ctx, "test-server", core.HealthHealthy)
	em.RecordHealthStatus(ctx, "test-server", core.HealthDegraded)
	em.RecordHealthStatus(ctx, "test-server", core.HealthUnhealthy)
}

func TestMapErrorCodeToGRPC(t *testing.T) {
	tests := []struct {
		code     errors.ErrorCode
		wantCode codes.Code
	}{
		{errors.CodeInvalidInput, codes.InvalidArgument},
		{errors.CodeNotFound, codes.NotFound},
		{errors.CodeUnauthorized, codes.Unauthenticated},
		{errors.CodeTimeout, codes.DeadlineExceeded},
		{errors.CodeRateLimit, codes.ResourceExhausted},
		{errors.CodeToolFailure, codes.FailedPrecondition},
		{errors.CodeLLMError, codes.Unavailable},
		{errors.CodeMemoryError, codes.DataLoss},
		{errors.CodeContextLost, codes.Canceled},
		{errors.CodeInternal, codes.Internal},
		{"UNKNOWN_CODE", codes.Unknown},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			got := mapErrorCodeToGRPC(tt.code)
			if got != tt.wantCode {
				t.Errorf("mapErrorCodeToGRPC(%v) = %v, want %v", tt.code, got, tt.wantCode)
			}
		})
	}
}
