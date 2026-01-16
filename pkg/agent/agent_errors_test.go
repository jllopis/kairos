// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"testing"
	"time"

	"github.com/jllopis/kairos/pkg/core"
	"github.com/jllopis/kairos/pkg/errors"
	"github.com/jllopis/kairos/pkg/llm"
)

func TestWrapLLMError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		model     string
		wantCode  errors.ErrorCode
		wantModel bool
	}{
		{
			name:      "nil error",
			err:       nil,
			model:     "gpt-4",
			wantCode:  "",
			wantModel: false,
		},
		{
			name:      "standard error",
			err:       errors.New(errors.CodeInternal, "test error", nil),
			model:     "gpt-4",
			wantCode:  errors.CodeLLMError,
			wantModel: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ke := WrapLLMError(tt.err, tt.model)
			if tt.err == nil {
				if ke != nil {
					t.Errorf("WrapLLMError() = %v, want nil", ke)
				}
				return
			}
			if ke == nil {
				t.Fatalf("WrapLLMError() = nil, want non-nil")
			}
			if ke.Code != tt.wantCode {
				t.Errorf("WrapLLMError().Code = %v, want %v", ke.Code, tt.wantCode)
			}
			if tt.wantModel {
				if ke.Context["model"] != tt.model {
					t.Errorf("WrapLLMError().Context[model] = %v, want %v", ke.Context["model"], tt.model)
				}
			}
			if !ke.Recoverable {
				t.Error("WrapLLMError().Recoverable = false, want true")
			}
		})
	}
}

func TestWrapToolError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		toolName   string
		toolCallID string
		wantCode   errors.ErrorCode
	}{
		{
			name:       "nil error",
			err:        nil,
			toolName:   "test-tool",
			toolCallID: "call-123",
			wantCode:   "",
		},
		{
			name:       "standard error",
			err:        errors.New(errors.CodeInternal, "test error", nil),
			toolName:   "test-tool",
			toolCallID: "call-123",
			wantCode:   errors.CodeToolFailure,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ke := WrapToolError(tt.err, tt.toolName, tt.toolCallID)
			if tt.err == nil {
				if ke != nil {
					t.Errorf("WrapToolError() = %v, want nil", ke)
				}
				return
			}
			if ke == nil {
				t.Fatalf("WrapToolError() = nil, want non-nil")
			}
			if ke.Code != tt.wantCode {
				t.Errorf("WrapToolError().Code = %v, want %v", ke.Code, tt.wantCode)
			}
			if ke.Context["tool_name"] != tt.toolName {
				t.Errorf("WrapToolError().Context[tool_name] = %v, want %v", ke.Context["tool_name"], tt.toolName)
			}
		})
	}
}

func TestWrapMemoryError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		operation string
		wantCode  errors.ErrorCode
	}{
		{
			name:      "nil error",
			err:       nil,
			operation: "store",
			wantCode:  "",
		},
		{
			name:      "standard error",
			err:       errors.New(errors.CodeInternal, "test error", nil),
			operation: "retrieve",
			wantCode:  errors.CodeMemoryError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ke := WrapMemoryError(tt.err, tt.operation)
			if tt.err == nil {
				if ke != nil {
					t.Errorf("WrapMemoryError() = %v, want nil", ke)
				}
				return
			}
			if ke == nil {
				t.Fatalf("WrapMemoryError() = nil, want non-nil")
			}
			if ke.Code != tt.wantCode {
				t.Errorf("WrapMemoryError().Code = %v, want %v", ke.Code, tt.wantCode)
			}
		})
	}
}

func TestWrapTimeoutError(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		operation     string
		maxIterations int
		wantCode      errors.ErrorCode
	}{
		{
			name:          "nil error",
			err:           nil,
			operation:     "agent-loop",
			maxIterations: 10,
			wantCode:      "",
		},
		{
			name:          "timeout error",
			err:           errors.New(errors.CodeTimeout, "test error", nil),
			operation:     "agent-loop",
			maxIterations: 10,
			wantCode:      errors.CodeTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ke := WrapTimeoutError(tt.err, tt.operation, tt.maxIterations)
			if tt.err == nil {
				if ke != nil {
					t.Errorf("WrapTimeoutError() = %v, want nil", ke)
				}
				return
			}
			if ke == nil {
				t.Fatalf("WrapTimeoutError() = nil, want non-nil")
			}
			if ke.Code != tt.wantCode {
				t.Errorf("WrapTimeoutError().Code = %v, want %v", ke.Code, tt.wantCode)
			}
			if ke.Recoverable {
				t.Error("WrapTimeoutError().Recoverable = true, want false")
			}
		})
	}
}

func TestNewInvalidInputError(t *testing.T) {
	ke := NewInvalidInputError("test message")
	if ke == nil {
		t.Fatal("NewInvalidInputError() = nil, want non-nil")
	}
	if ke.Code != errors.CodeInvalidInput {
		t.Errorf("NewInvalidInputError().Code = %v, want %v", ke.Code, errors.CodeInvalidInput)
	}
	if ke.Message != "test message" {
		t.Errorf("NewInvalidInputError().Message = %v, want %v", ke.Message, "test message")
	}
	if ke.Recoverable {
		t.Error("NewInvalidInputError().Recoverable = true, want false")
	}
}

func TestNewNotFoundError(t *testing.T) {
	ke := NewNotFoundError("task", "task-123")
	if ke == nil {
		t.Fatal("NewNotFoundError() = nil, want non-nil")
	}
	if ke.Code != errors.CodeNotFound {
		t.Errorf("NewNotFoundError().Code = %v, want %v", ke.Code, errors.CodeNotFound)
	}
	if ke.Context["resource"] != "task" {
		t.Errorf("NewNotFoundError().Context[resource] = %v, want %v", ke.Context["resource"], "task")
	}
	if ke.Context["name"] != "task-123" {
		t.Errorf("NewNotFoundError().Context[name] = %v, want %v", ke.Context["name"], "task-123")
	}
}

func TestErrorMetricsIntegration(t *testing.T) {
	ctx := context.Background()

	// Initialize error metrics
	em := InitErrorMetrics(ctx)
	if em == nil {
		t.Fatal("InitErrorMetrics() = nil, want non-nil")
	}

	// Get global metrics
	globalEM := GetErrorMetrics()
	if globalEM == nil {
		t.Fatal("GetErrorMetrics() = nil, want non-nil")
	}

	// Test RecordError (should not panic even with nil metrics)
	testErr := errors.New(errors.CodeToolFailure, "test", nil)
	em.RecordError(ctx, testErr, "test-component")

	// Test RecordRecovery
	em.RecordRecovery(ctx, errors.CodeToolFailure)

	// Test RecordHealthStatus
	em.RecordHealthStatus(ctx, "test-component", core.HealthHealthy)
	em.RecordHealthStatus(ctx, "test-component", core.HealthDegraded)
	em.RecordHealthStatus(ctx, "test-component", core.HealthUnhealthy)

	// Test RecordCircuitBreakerState
	em.RecordCircuitBreakerState(ctx, "test-component", 2) // CLOSED
	em.RecordCircuitBreakerState(ctx, "test-component", 1) // HALF_OPEN
	em.RecordCircuitBreakerState(ctx, "test-component", 0) // OPEN
}

func TestAgentHealthChecker(t *testing.T) {
	// Create a mock agent
	a := &Agent{
		id:  "test-agent",
		llm: &llm.MockProvider{Response: "test"},
	}

	checker := NewAgentHealthChecker(a)
	if checker == nil {
		t.Fatal("NewAgentHealthChecker() = nil, want non-nil")
	}

	ctx := context.Background()
	result := checker.Check(ctx)

	if result.Component != "agent:test-agent" {
		t.Errorf("Check().Component = %v, want agent:test-agent", result.Component)
	}
	if result.Status != core.HealthHealthy {
		t.Errorf("Check().Status = %v, want %v", result.Status, core.HealthHealthy)
	}
	if result.LastCheck.IsZero() {
		t.Error("Check().LastCheck is zero, want non-zero")
	}
}

func TestAgentHealthChecker_NoLLM(t *testing.T) {
	a := &Agent{
		id: "test-agent",
		// No LLM provider
	}

	checker := NewAgentHealthChecker(a)
	ctx := context.Background()
	result := checker.Check(ctx)

	if result.Status != core.HealthUnhealthy {
		t.Errorf("Check().Status = %v, want %v", result.Status, core.HealthUnhealthy)
	}
	if result.Message != "LLM provider not configured" {
		t.Errorf("Check().Message = %v, want 'LLM provider not configured'", result.Message)
	}
}

func TestLLMHealthChecker(t *testing.T) {
	checkCalled := false
	checker := NewLLMHealthChecker("test-llm", func(ctx context.Context) error {
		checkCalled = true
		return nil
	})

	ctx := context.Background()
	result := checker.Check(ctx)

	if !checkCalled {
		t.Error("checkFunc was not called")
	}
	if result.Status != core.HealthHealthy {
		t.Errorf("Check().Status = %v, want %v", result.Status, core.HealthHealthy)
	}
}

func TestLLMHealthChecker_NoCheckFunc(t *testing.T) {
	checker := NewLLMHealthChecker("test-llm", nil)

	ctx := context.Background()
	result := checker.Check(ctx)

	if result.Status != core.HealthHealthy {
		t.Errorf("Check().Status = %v, want %v", result.Status, core.HealthHealthy)
	}
}

func TestLLMHealthChecker_CacheTTL(t *testing.T) {
	callCount := 0
	checker := NewLLMHealthChecker("test-llm", func(ctx context.Context) error {
		callCount++
		return nil
	})
	checker.minInterval = 100 * time.Millisecond

	ctx := context.Background()

	// First call
	checker.Check(ctx)
	if callCount != 1 {
		t.Errorf("callCount = %d, want 1", callCount)
	}

	// Second call (should use cache)
	checker.Check(ctx)
	if callCount != 1 {
		t.Errorf("callCount = %d, want 1 (cached)", callCount)
	}

	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)
	checker.Check(ctx)
	if callCount != 2 {
		t.Errorf("callCount = %d, want 2 (after cache expire)", callCount)
	}
}

func TestMemoryHealthChecker(t *testing.T) {
	mem := &mockMemory{}
	checker := NewMemoryHealthChecker("test-memory", mem)

	ctx := context.Background()
	result := checker.Check(ctx)

	if result.Status != core.HealthHealthy {
		t.Errorf("Check().Status = %v, want %v", result.Status, core.HealthHealthy)
	}
}

func TestMemoryHealthChecker_NoMemory(t *testing.T) {
	checker := NewMemoryHealthChecker("test-memory", nil)

	ctx := context.Background()
	result := checker.Check(ctx)

	if result.Status != core.HealthUnhealthy {
		t.Errorf("Check().Status = %v, want %v", result.Status, core.HealthUnhealthy)
	}
}

func TestMCPHealthChecker(t *testing.T) {
	checker := NewMCPHealthChecker("test-mcp", func(ctx context.Context) (int, error) {
		return 5, nil
	})

	ctx := context.Background()
	result := checker.Check(ctx)

	if result.Status != core.HealthHealthy {
		t.Errorf("Check().Status = %v, want %v", result.Status, core.HealthHealthy)
	}
}

func TestMCPHealthChecker_NoCheckFunc(t *testing.T) {
	checker := NewMCPHealthChecker("test-mcp", nil)

	ctx := context.Background()
	result := checker.Check(ctx)

	if result.Status != core.HealthHealthy {
		t.Errorf("Check().Status = %v, want %v", result.Status, core.HealthHealthy)
	}
}

// Mock implementations for testing

type mockMemory struct{}

func (m *mockMemory) Store(ctx context.Context, data any) error {
	return nil
}

func (m *mockMemory) Retrieve(ctx context.Context, query any) (any, error) {
	return nil, nil
}
