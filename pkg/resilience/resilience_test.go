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

func TestRetrySuccess(t *testing.T) {
	attempts := 0
	config := DefaultRetryConfig()
	err := config.Do(context.Background(), func() error {
		attempts++
		if attempts < 3 {
			return errors.New("transient error")
		}
		return nil
	})

	if err != nil {
		t.Errorf("expected success, got error: %v", err)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestRetryMaxAttemptsExceeded(t *testing.T) {
	attempts := 0
	config := DefaultRetryConfig().WithMaxAttempts(2)
	err := config.Do(context.Background(), func() error {
		attempts++
		return errors.New("always fails")
	})

	if err == nil {
		t.Errorf("expected error after max attempts")
	}
	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}

func TestRetryNonRecoverable(t *testing.T) {
	attempts := 0
	config := DefaultRetryConfig().WithIsRecoverable(func(err error) bool {
		return false // Never recoverable
	})
	err := config.Do(context.Background(), func() error {
		attempts++
		return errors.New("non-recoverable error")
	})

	if err == nil {
		t.Errorf("expected error")
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", attempts)
	}
}

func TestRetryContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	config := DefaultRetryConfig().WithInitialDelay(100 * time.Millisecond)

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	attempts := 0
	err := config.Do(ctx, func() error {
		attempts++
		return errors.New("transient error")
	})

	if err == nil {
		t.Errorf("expected context error")
	}
	if attempts < 1 {
		t.Errorf("expected at least 1 attempt, got %d", attempts)
	}
}

func TestRetryWithResult(t *testing.T) {
	attempts := 0
	config := DefaultRetryConfig()
	result, err := config.DoWithResult(context.Background(), func() (interface{}, error) {
		attempts++
		if attempts < 2 {
			return nil, errors.New("transient")
		}
		return "success", nil
	})

	if err != nil {
		t.Errorf("expected success, got error: %v", err)
	}
	if result != "success" {
		t.Errorf("expected 'success', got %v", result)
	}
	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}

func TestCircuitBreakerClosed(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 3,
		Name:             "test",
	})

	if cb.State() != StateClosed {
		t.Errorf("expected initial state Closed")
	}

	// Successful calls should keep it closed
	for i := 0; i < 5; i++ {
		err := cb.Call(context.Background(), func() error { return nil })
		if err != nil {
			t.Errorf("call %d failed: %v", i, err)
		}
	}

	if cb.State() != StateClosed {
		t.Errorf("expected state to remain Closed after success")
	}
}

func TestCircuitBreakerOpen(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 2,
		Name:             "test",
	})

	// Trigger failures to open the circuit
	for i := 0; i < 2; i++ {
		_ = cb.Call(context.Background(), func() error {
			return errors.New("failure")
		})
	}

	if cb.State() != StateOpen {
		t.Errorf("expected state Open after %d failures", 2)
	}

	// Subsequent calls should be rejected
	err := cb.Call(context.Background(), func() error {
		t.Fatalf("should not execute in open state")
		return nil
	})

	if err == nil {
		t.Errorf("expected error when circuit is open")
	}

	// Error should indicate circuit is open
	if ke, ok := err.(*kerrors.KairosError); ok && !ke.Recoverable {
		t.Errorf("expected circuit breaker error to be marked recoverable")
	}
}

func TestCircuitBreakerHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 1,
		SuccessThreshold: 2,
		Timeout:          100 * time.Millisecond,
		Name:             "test",
	})

	// Open the circuit
	_ = cb.Call(context.Background(), func() error { return errors.New("fail") })
	if cb.State() != StateOpen {
		t.Fatalf("expected circuit to be open")
	}

	// Wait for timeout to transition to half-open
	time.Sleep(150 * time.Millisecond)
	_ = cb.Call(context.Background(), func() error { return nil })

	if cb.State() != StateHalfOpen {
		t.Errorf("expected state HalfOpen after timeout")
	}

	// Another success should close it
	_ = cb.Call(context.Background(), func() error { return nil })

	if cb.State() != StateClosed {
		t.Errorf("expected state Closed after successes in half-open")
	}
}

func TestCircuitBreakerReset(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 1,
		Name:             "test",
	})

	// Open the circuit
	_ = cb.Call(context.Background(), func() error { return errors.New("fail") })

	if cb.State() != StateOpen {
		t.Fatalf("expected circuit to be open")
	}

	// Reset should go back to closed
	cb.Reset()

	if cb.State() != StateClosed {
		t.Errorf("expected state Closed after reset")
	}

	// Calls should succeed
	err := cb.Call(context.Background(), func() error { return nil })
	if err != nil {
		t.Errorf("call failed after reset: %v", err)
	}
}

func TestKairosErrorRecoverable(t *testing.T) {
	ke := kerrors.New(kerrors.CodeTimeout, "timed out", nil).WithRecoverable(true)

	config := DefaultRetryConfig()
	attempts := 0
	err := config.Do(context.Background(), func() error {
		attempts++
		if attempts < 2 {
			return ke
		}
		return nil
	})

	if err != nil {
		t.Errorf("expected retry to succeed")
	}
	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}
