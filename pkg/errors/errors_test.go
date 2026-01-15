// SPDX-License-Identifier: Apache-2.0
// Package errors provides typed error handling with rich context for Kairos.
// See docs/ERROR_HANDLING.md for strategy and examples.
package errors

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestNew(t *testing.T) {
	cause := errors.New("network timeout")
	ke := New(CodeTimeout, "tool execution timed out", cause)

	if ke.Code != CodeTimeout {
		t.Errorf("expected CodeTimeout, got %v", ke.Code)
	}
	if ke.Message != "tool execution timed out" {
		t.Errorf("expected message 'tool execution timed out', got %q", ke.Message)
	}
	if ke.Err != cause {
		t.Errorf("expected cause to be preserved")
	}
	if !errors.Is(ke, cause) {
		t.Errorf("expected errors.Is to work with wrapped error")
	}
}

func TestWithContext(t *testing.T) {
	ke := New(CodeToolFailure, "tool failed", nil)
	ke.WithContext("tool", "get_weather").
		WithContext("args", map[string]interface{}{"city": "London"})

	if ke.Context["tool"] != "get_weather" {
		t.Errorf("expected context tool to be 'get_weather'")
	}
	if ke.Context["args"] == nil {
		t.Errorf("expected context args to be set")
	}
}

func TestWithAttribute(t *testing.T) {
	ke := New(CodeToolFailure, "tool failed", nil)
	ke.WithAttribute("tool_name", "get_weather").
		WithAttribute("retry_count", "3")

	if ke.Attributes["tool_name"] != "get_weather" {
		t.Errorf("expected attribute tool_name")
	}
	if ke.Attributes["retry_count"] != "3" {
		t.Errorf("expected attribute retry_count")
	}
}

func TestWithRecoverable(t *testing.T) {
	ke := New(CodeToolFailure, "network error", nil)
	if ke.Recoverable {
		t.Errorf("expected recoverable to be false by default")
	}

	ke.WithRecoverable(true)
	if !ke.Recoverable {
		t.Errorf("expected recoverable to be true after WithRecoverable")
	}
}

func TestError(t *testing.T) {
	tests := []struct {
		name     string
		ke       *KairosError
		expected string
	}{
		{
			name:     "with cause",
			ke:       New(CodeTimeout, "operation timed out", errors.New("deadline exceeded")),
			expected: "[TIMEOUT] operation timed out: deadline exceeded",
		},
		{
			name:     "without cause",
			ke:       New(CodeNotFound, "tool not found", nil),
			expected: "[NOT_FOUND] tool not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ke.Error()
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestAsKairosError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected ErrorCode
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: "",
		},
		{
			name:     "already KairosError",
			err:      New(CodeToolFailure, "failed", nil),
			expected: CodeToolFailure,
		},
		{
			name:     "generic error",
			err:      errors.New("generic error"),
			expected: CodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ke := AsKairosError(tt.err)
			if tt.expected == "" {
				if ke != nil {
					t.Errorf("expected nil for nil error")
				}
			} else {
				if ke == nil {
					t.Errorf("expected non-nil KairosError")
				} else if ke.Code != tt.expected {
					t.Errorf("expected %v, got %v", tt.expected, ke.Code)
				}
			}
		})
	}
}

func TestMarshalJSON(t *testing.T) {
	ke := New(CodeToolFailure, "tool failed", errors.New("network error"))
	ke.WithContext("tool", "get_weather").
		WithAttribute("retry_count", "1").
		WithRecoverable(true)

	data, err := json.Marshal(ke)
	if err != nil {
		t.Fatalf("unexpected error marshaling: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unexpected error unmarshaling: %v", err)
	}

	if result["code"] != "TOOL_FAILURE" {
		t.Errorf("expected code 'TOOL_FAILURE', got %v", result["code"])
	}
	if result["recoverable"] != true {
		t.Errorf("expected recoverable true")
	}
}

func TestStatusCode(t *testing.T) {
	tests := []struct {
		code     ErrorCode
		expected int
	}{
		{CodeNotFound, 404},
		{CodeUnauthorized, 401},
		{CodeInvalidInput, 400},
		{CodeTimeout, 408},
		{CodeRateLimit, 429},
		{CodeInternal, 500},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			ke := New(tt.code, "test", nil)
			if ke.StatusCode != tt.expected {
				t.Errorf("expected status %d, got %d", tt.expected, ke.StatusCode)
			}
		})
	}
}
