// SPDX-License-Identifier: Apache-2.0
// Example demonstrating production-grade error handling with Kairos.
// This example shows how to use typed errors, resilience patterns, and OTEL integration.
// See docs/ERROR_HANDLING.md for detailed strategy.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jllopis/kairos/pkg/errors"
	"github.com/jllopis/kairos/pkg/resilience"
	"github.com/jllopis/kairos/pkg/telemetry"
	"go.opentelemetry.io/otel"
)

func main() {
	// Initialize telemetry
	shutdown, err := telemetry.Init("error-handling-example", "1.0.0")
	if err != nil {
		slog.Error("failed to initialize telemetry", "error", err)
		return
	}
	defer shutdown(context.Background())

	ctx := context.Background()
	tracer := otel.Tracer("example")
	_, span := tracer.Start(ctx, "ExampleErrorHandling")
	defer span.End()

	// Example 1: Creating typed errors
	fmt.Println("=== Example 1: Typed Errors ===")
	toolErr := errors.New(errors.CodeToolFailure, "tool execution failed", fmt.Errorf("network timeout")).
		WithContext("tool", "get_weather").
		WithContext("city", "London").
		WithAttribute("retry_count", "2").
		WithRecoverable(true)

	fmt.Printf("Error: %v\n", toolErr)
	fmt.Printf("Code: %v, Recoverable: %v\n", toolErr.Code, toolErr.Recoverable)
	fmt.Printf("Context: %v\n", toolErr.Context)
	fmt.Printf("Status Code: %d\n\n", toolErr.StatusCode)

	// Record in OTEL trace
	telemetry.RecordError(span, toolErr)

	// Example 2: Retry with configuration
	fmt.Println("=== Example 2: Retry Pattern ===")
	retryConfig := resilience.DefaultRetryConfig().
		WithMaxAttempts(3).
		WithInitialDelay(100 * time.Millisecond).
		WithIsRecoverable(func(err error) bool {
			if ke, ok := err.(*errors.KairosError); ok {
				return ke.Recoverable
			}
			return false
		})

	attempts := 0
	err = retryConfig.Do(ctx, func() error {
		attempts++
		fmt.Printf("Attempt %d\n", attempts)
		if attempts < 2 {
			// Simulate recoverable error
			return errors.New(errors.CodeTimeout, "operation timed out", nil).
				WithRecoverable(true)
		}
		fmt.Println("Success!")
		return nil
	})

	if err != nil {
		fmt.Printf("Failed after retries: %v\n", err)
	}
	fmt.Printf("Total attempts: %d\n\n", attempts)

	// Example 3: Circuit breaker pattern
	fmt.Println("=== Example 3: Circuit Breaker ===")
	cb := resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
		FailureThreshold: 2,
		SuccessThreshold: 1,
		Timeout:          100 * time.Millisecond,
		Name:             "example_breaker",
	})

	fmt.Printf("Initial state: %v\n", cb.State())

	// Trigger failures to open circuit
	for i := 0; i < 3; i++ {
		err := cb.Call(ctx, func() error {
			return errors.New(errors.CodeInternal, "service error", nil)
		})
		if err != nil {
			fmt.Printf("Call %d: Error - %v\n", i+1, err)
		} else {
			fmt.Printf("Call %d: Success\n", i+1)
		}
	}

	fmt.Printf("State after failures: %v\n", cb.State())

	// Wait for timeout to transition to half-open
	fmt.Println("Waiting for timeout...")
	time.Sleep(150 * time.Millisecond)

	// Try successful call in half-open state
	err = cb.Call(ctx, func() error {
		fmt.Println("Half-open: attempting recovery")
		return nil
	})
	if err == nil {
		fmt.Println("Half-open: recovery successful")
	}

	fmt.Printf("Final state: %v\n\n", cb.State())

	// Example 4: Error classification for monitoring
	fmt.Println("=== Example 4: Error Classification ===")
	errorExamples := []struct {
		name string
		err  error
	}{
		{"Tool failure", errors.New(errors.CodeToolFailure, "tool failed", nil).WithRecoverable(true)},
		{"Timeout", errors.New(errors.CodeTimeout, "operation timed out", nil).WithRecoverable(true)},
		{"Invalid input", errors.New(errors.CodeInvalidInput, "invalid tool parameter", nil).WithRecoverable(false)},
		{"Not found", errors.New(errors.CodeNotFound, "tool not found", nil).WithRecoverable(false)},
		{"LLM error", errors.New(errors.CodeLLMError, "LLM provider error", nil).WithRecoverable(true)},
	}

	for _, ex := range errorExamples {
		if ke, ok := ex.err.(*errors.KairosError); ok {
			fmt.Printf("%-15s Code: %-20s Recoverable: %v StatusCode: %d\n",
				ex.name, ke.Code, ke.Recoverable, ke.StatusCode)
		}
	}

	fmt.Println("\n=== All examples completed ===")
}
