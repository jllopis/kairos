// SPDX-License-Identifier: Apache-2.0
// Package resilience provides retry and circuit breaker patterns for Kairos.
// See docs/ERROR_HANDLING.md for strategy and examples.
package resilience

import (
	"context"
	"errors"
	"testing"
	"time"

	kerrors "github.com/jllopis/kairos/pkg/errors"
)

func TestWithTimeout(t *testing.T) {
	tests := []struct {
		name        string
		duration    time.Duration
		sleepTime   time.Duration
		expectError bool
	}{
		{"fast operation", 1 * time.Second, 10 * time.Millisecond, false},
		{"slow operation", 50 * time.Millisecond, 200 * time.Millisecond, true},
		{"no timeout", 0, 100 * time.Millisecond, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := TimeoutConfig{Duration: tt.duration, ErrorOnTimeout: true}
			err := WithTimeout(context.Background(), config, func() error {
				time.Sleep(tt.sleepTime)
				return nil
			})

			if tt.expectError {
				if err == nil {
					t.Errorf("expected timeout error")
				}
				if ke, ok := err.(*kerrors.KairosError); ok {
					if ke.Code != kerrors.CodeTimeout {
						t.Errorf("expected CodeTimeout, got %v", ke.Code)
					}
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestWithTimeoutResult(t *testing.T) {
	config := TimeoutConfig{Duration: 1 * time.Second}

	value, err := WithTimeoutResult(context.Background(), config, func() (interface{}, error) {
		return "success", nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if value != "success" {
		t.Errorf("expected 'success', got %v", value)
	}
}

func TestWithTimeoutResultTimeout(t *testing.T) {
	config := TimeoutConfig{Duration: 50 * time.Millisecond}

	value, err := WithTimeoutResult(context.Background(), config, func() (interface{}, error) {
		time.Sleep(200 * time.Millisecond)
		return "success", nil
	})

	if err == nil {
		t.Errorf("expected timeout error")
	}
	if value != nil {
		t.Errorf("expected nil value on timeout")
	}
}

func TestStaticFallback(t *testing.T) {
	fallback := &StaticFallback{Value: "default"}

	value, err := fallback.Execute(context.Background(), errors.New("primary failed"))

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if value != "default" {
		t.Errorf("expected 'default', got %v", value)
	}
}

func TestErrorFallback(t *testing.T) {
	fallback := &ErrorFallback{Message: "all attempts failed"}

	value, err := fallback.Execute(context.Background(), errors.New("primary failed"))

	if err == nil {
		t.Errorf("expected error")
	}
	if value != nil {
		t.Errorf("expected nil value")
	}
}

func TestCachedFallback(t *testing.T) {
	fallback := &CachedFallback{Cache: "cached_value"}

	value, err := fallback.Execute(context.Background(), errors.New("primary failed"))

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if value != "cached_value" {
		t.Errorf("expected 'cached_value', got %v", value)
	}
}

func TestCachedFallbackEmpty(t *testing.T) {
	fallback := &CachedFallback{Cache: nil}

	value, err := fallback.Execute(context.Background(), errors.New("primary failed"))

	if err == nil {
		t.Errorf("expected error when cache is empty")
	}
	if value != nil {
		t.Errorf("expected nil value")
	}
}

func TestChainedFallback(t *testing.T) {
	fallback := &ChainedFallback{
		Fallbacks: []FallbackStrategy{
			&ErrorFallback{Message: "first failed"},
			&ErrorFallback{Message: "second failed"},
			&StaticFallback{Value: "final fallback"},
		},
	}

	value, err := fallback.Execute(context.Background(), errors.New("primary failed"))

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if value != "final fallback" {
		t.Errorf("expected 'final fallback', got %v", value)
	}
}

func TestWithFallback(t *testing.T) {
	fallback := &StaticFallback{Value: "default"}

	value, err := WithFallback(context.Background(),
		func() (interface{}, error) {
			return nil, errors.New("primary failed")
		},
		fallback,
	)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if value != "default" {
		t.Errorf("expected 'default', got %v", value)
	}
}

func TestWithFallbackSuccess(t *testing.T) {
	fallback := &StaticFallback{Value: "default"}

	value, err := WithFallback(context.Background(),
		func() (interface{}, error) {
			return "primary", nil
		},
		fallback,
	)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if value != "primary" {
		t.Errorf("expected 'primary', got %v", value)
	}
}

func TestGracefulDegradation(t *testing.T) {
	calls := 0
	gd := &GracefulDegradation{
		Primary: func() (interface{}, error) {
			calls++
			if calls < 3 {
				return nil, errors.New("temporarily unavailable")
			}
			return "success", nil
		},
		Fallback:  &StaticFallback{Value: "degraded"},
		MaxErrors: 2,
	}

	// First call fails, errorCount becomes 1, still operational
	_, err := gd.Execute(context.Background())
	if err == nil {
		t.Errorf("expected error on first call")
	}
	if !gd.IsOperational() {
		t.Errorf("should still be operational after 1 error (max 2)")
	}

	// Second call fails, errorCount becomes 2, should use fallback (not degraded)
	value, err := gd.Execute(context.Background())
	if err != nil {
		t.Errorf("expected fallback to succeed, got error: %v", err)
	}
	if value != "degraded" {
		t.Errorf("expected 'degraded', got %v", value)
	}
	if gd.IsOperational() {
		t.Errorf("should be degraded after reaching MaxErrors")
	}

	// Third call: Primary succeeds, error count resets
	value, err = gd.Execute(context.Background())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if value != "success" {
		t.Errorf("expected 'success', got %v", value)
	}
	if !gd.IsOperational() {
		t.Errorf("should be operational after primary succeeds")
	}
}

func TestGracefulDegradationStatus(t *testing.T) {
	gd := &GracefulDegradation{
		Primary: func() (interface{}, error) {
			return nil, errors.New("error")
		},
		Fallback:  &StaticFallback{Value: "fallback"},
		MaxErrors: 1,
	}

	if gd.Status() != "operational" {
		t.Errorf("expected operational status initially")
	}

	gd.Execute(context.Background())
	if gd.Status() != "degraded" {
		t.Errorf("expected degraded status after error")
	}
}

func TestFallbackFunc(t *testing.T) {
	fallback := FallbackFunc(func(ctx context.Context, err error) (interface{}, error) {
		return "recovered", nil
	})

	value, err := fallback.Execute(context.Background(), errors.New("primary failed"))

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if value != "recovered" {
		t.Errorf("expected 'recovered', got %v", value)
	}
}
