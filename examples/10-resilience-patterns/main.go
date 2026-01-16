// SPDX-License-Identifier: Apache-2.0
// Example demonstrating Phase 2 resilience patterns for Kairos.
// This example shows health checks, timeouts, and graceful degradation.
// See docs/ERROR_HANDLING.md for detailed strategy.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jllopis/kairos/pkg/core"
	"github.com/jllopis/kairos/pkg/resilience"
	"github.com/jllopis/kairos/pkg/telemetry"
	"go.opentelemetry.io/otel"
)

// SimulatedLLMProvider simulates an LLM service that can be healthy or degraded.
type SimulatedLLMProvider struct {
	failureCount int
	maxFailures  int
}

// Check implements HealthChecker.
func (s *SimulatedLLMProvider) Check(ctx context.Context) core.HealthResult {
	if s.failureCount >= s.maxFailures {
		return core.HealthResult{
			Status:    core.HealthUnhealthy,
			Message:   fmt.Sprintf("failed %d times", s.failureCount),
			LastCheck: time.Now(),
		}
	}

	// Simulate 50% chance of degradation
	if s.failureCount > 0 {
		return core.HealthResult{
			Status:    core.HealthDegraded,
			Message:   fmt.Sprintf("recovering, %d failures", s.failureCount),
			LastCheck: time.Now(),
		}
	}

	return core.HealthResult{
		Status:    core.HealthHealthy,
		Message:   "operational",
		LastCheck: time.Now(),
	}
}

func main() {
	// Initialize telemetry
	shutdown, err := telemetry.Init("resilience-phase2-example", "1.0.0")
	if err != nil {
		slog.Error("failed to initialize telemetry", "error", err)
		return
	}
	defer shutdown(context.Background())

	ctx := context.Background()
	tracer := otel.Tracer("example")
	_, span := tracer.Start(ctx, "Phase2Example")
	defer span.End()

	fmt.Println("=== Phase 2: Resilience Patterns ===")

	// Example 1: Health Checks
	fmt.Println("--- Example 1: Health Checks ---")
	provider := core.NewDefaultHealthCheckProvider(10 * time.Second)

	llmChecker := &SimulatedLLMProvider{failureCount: 0}
	provider.RegisterChecker("llm", llmChecker)
	provider.RegisterChecker("memory", core.NewSimpleHealthChecker(core.HealthHealthy, "in-memory storage"))
	provider.RegisterChecker("cache", core.NewSimpleHealthChecker(core.HealthDegraded, "cache eviction rate high"))

	results, overallStatus := provider.CheckAll(ctx)
	fmt.Printf("Overall Status: %v\n", overallStatus)
	for _, result := range results {
		fmt.Printf("  %s: %s (%s)\n", result.Component, result.Status, result.Message)
	}
	fmt.Println()

	// Example 2: Timeout Boundaries
	fmt.Println("--- Example 2: Timeout Boundaries ---")

	// Fast operation
	config := resilience.TimeoutConfig{Duration: 1 * time.Second}
	err = resilience.WithTimeout(ctx, config, func() error {
		fmt.Println("Fast operation: starting")
		time.Sleep(100 * time.Millisecond)
		fmt.Println("Fast operation: completed")
		return nil
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	// Slow operation (times out)
	fmt.Println("\nSlow operation: starting (will timeout)")
	config = resilience.TimeoutConfig{Duration: 50 * time.Millisecond}
	err = resilience.WithTimeout(ctx, config, func() error {
		time.Sleep(200 * time.Millisecond)
		return nil
	})
	if err != nil {
		fmt.Printf("Expected timeout: %v\n", err)
	}
	fmt.Println()

	// Example 3: Fallback Strategies
	fmt.Println("--- Example 3: Fallback Strategies ---")

	// Static fallback
	fmt.Println("Static Fallback:")
	staticFallback := &resilience.StaticFallback{Value: "default_response"}
	value, _ := resilience.WithFallback(ctx,
		func() (interface{}, error) {
			return nil, fmt.Errorf("primary service unavailable")
		},
		staticFallback,
	)
	fmt.Printf("  Result: %v\n", value)

	// Cached fallback
	fmt.Println("\nCached Fallback:")
	cachedFallback := &resilience.CachedFallback{Cache: "last_known_good_data"}
	value, _ = resilience.WithFallback(ctx,
		func() (interface{}, error) {
			return nil, fmt.Errorf("database unavailable")
		},
		cachedFallback,
	)
	fmt.Printf("  Result: %v\n", value)

	// Chained fallback
	fmt.Println("\nChained Fallback (try multiple strategies):")
	chainedFallback := &resilience.ChainedFallback{
		Fallbacks: []resilience.FallbackStrategy{
			&resilience.ErrorFallback{Message: "primary unavailable"},
			&resilience.ErrorFallback{Message: "secondary unavailable"},
			&resilience.StaticFallback{Value: "finally using default"},
		},
	}
	value, _ = resilience.WithFallback(ctx,
		func() (interface{}, error) {
			return nil, fmt.Errorf("all services down")
		},
		chainedFallback,
	)
	fmt.Printf("  Result: %v\n", value)
	fmt.Println()

	// Example 4: Graceful Degradation
	fmt.Println("--- Example 4: Graceful Degradation ---")

	callCount := 0
	gd := &resilience.GracefulDegradation{
		Primary: func() (interface{}, error) {
			callCount++
			if callCount <= 2 {
				return nil, fmt.Errorf("service overloaded")
			}
			return "normal_operation", nil
		},
		Fallback: &resilience.StaticFallback{Value: "degraded_mode"},
		MaxErrors: 2,
		LogError: func(err error) {
			fmt.Printf("  Error logged: %v\n", err)
		},
	}

	for i := 0; i < 4; i++ {
		fmt.Printf("Call %d: ", i+1)
		value, err := gd.Execute(ctx)
		if err != nil {
			fmt.Printf("Error (operational: %v)\n", gd.IsOperational())
		} else {
			fmt.Printf("Result: %v (status: %s)\n", value, gd.Status())
		}
	}
	fmt.Println()

	// Example 5: Health-Based Retry
	fmt.Println("--- Example 5: Health-Based Retry ---")

	// Simulate LLM provider failure
	llmChecker.failureCount = 5
	result, _ := provider.Check(ctx, "llm")
	fmt.Printf("LLM Health: %v (%s)\n", result.Status, result.Message)

	// Decide whether to retry based on health
	if result.Status == core.HealthUnhealthy {
		fmt.Println("Service unhealthy - using fallback instead of retry")
		value, _ := resilience.WithFallback(ctx,
			func() (interface{}, error) {
				return nil, fmt.Errorf("llm service down")
			},
			&resilience.StaticFallback{Value: "using cached responses"},
		)
		fmt.Printf("Fallback Result: %v\n", value)
	} else if result.Status == core.HealthDegraded {
		fmt.Println("Service degraded - retrying with backoff")
	}

	fmt.Println("\n=== Phase 2 Examples Completed ===")
}
