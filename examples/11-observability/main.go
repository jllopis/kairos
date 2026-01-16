// SPDX-License-Identifier: Apache-2.0
// Example demonstrating Phase 3 observability patterns for Kairos.
// This example shows error metrics, health monitoring, and alerting integration.
// See docs/ERROR_HANDLING.md and docs/internal/observability-dashboards.go for detailed patterns.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/jllopis/kairos/pkg/errors"
	"github.com/jllopis/kairos/pkg/resilience"
	"github.com/jllopis/kairos/pkg/telemetry"
	"go.opentelemetry.io/otel"
)

// ErrorSimulator generates various error patterns for dashboard demonstration.
type ErrorSimulator struct {
	errorCount    int
	recoveryCount int
	mu            sync.Mutex
}

func (es *ErrorSimulator) recordError() {
	es.mu.Lock()
	es.errorCount++
	es.mu.Unlock()
}

func (es *ErrorSimulator) recordRecovery() {
	es.mu.Lock()
	es.recoveryCount++
	es.mu.Unlock()
}

func (es *ErrorSimulator) getStats() (int, int, float64) {
	es.mu.Lock()
	defer es.mu.Unlock()

	recovery := 0.0
	if es.errorCount > 0 {
		recovery = float64(es.recoveryCount) / float64(es.errorCount) * 100
	}
	return es.errorCount, es.recoveryCount, recovery
}

func main() {
	// Initialize telemetry with metrics
	shutdown, err := telemetry.Init("observability-phase3-example", "1.0.0")
	if err != nil {
		slog.Error("failed to initialize telemetry", "error", err)
		return
	}
	defer shutdown(context.Background())

	ctx := context.Background()
	tracer := otel.Tracer("example")
	_, span := tracer.Start(ctx, "Phase3Example")
	defer span.End()

	// Initialize error metrics
	metrics, err := telemetry.NewErrorMetrics(ctx)
	if err != nil {
		slog.Error("failed to create error metrics", "error", err)
		return
	}

	fmt.Println("=== Phase 3: Observability & Monitoring ===")

	// Example 1: Error Rate Tracking
	fmt.Println("--- Example 1: Error Rate Tracking ---")
	simulator := &ErrorSimulator{}

	errorPatterns := []struct {
		code      errors.ErrorCode
		message   string
		component string
		count     int
	}{
		{errors.CodeLLMError, "model overloaded", "llm-service", 3},
		{errors.CodeTimeout, "request timeout", "api-client", 2},
		{errors.CodeToolFailure, "tool execution failed", "tool-executor", 4},
		{errors.CodeRateLimit, "rate limit exceeded", "gateway", 1},
	}

	fmt.Println("Recording errors by type:")
	for _, pattern := range errorPatterns {
		for i := 0; i < pattern.count; i++ {
			err := errors.New(pattern.code, pattern.message, nil).
				WithContext("component", pattern.component).
				WithRecoverable(true)
			metrics.RecordErrorMetric(ctx, err, pattern.component)
			simulator.recordError()
		}
		fmt.Printf("  %s: %d errors\n", pattern.code, pattern.count)
	}

	errorCnt, _, _ := simulator.getStats()
	fmt.Printf("\nTotal errors recorded: %d\n\n", errorCnt)

	// Example 2: Recovery Tracking
	fmt.Println("--- Example 2: Recovery Tracking ---")
	fmt.Println("Recording successful recoveries:")

	recoveryPatterns := []struct {
		code  errors.ErrorCode
		count int
	}{
		{errors.CodeLLMError, 2},       // 2 of 3 recovered
		{errors.CodeTimeout, 2},        // 2 of 2 recovered
		{errors.CodeToolFailure, 3},    // 3 of 4 recovered
		{errors.CodeRateLimit, 1},      // 1 of 1 recovered
	}

	for _, pattern := range recoveryPatterns {
		for i := 0; i < pattern.count; i++ {
			metrics.RecordRecovery(ctx, pattern.code)
			simulator.recordRecovery()
		}
		fmt.Printf("  %s: %d recoveries\n", pattern.code, pattern.count)
	}

	_, recoveryCnt, recoveryRate := simulator.getStats()
	fmt.Printf("\nTotal errors: %d, Recoveries: %d, Recovery Rate: %.1f%%\n\n", errorCnt, recoveryCnt, recoveryRate)

	// Example 3: Error Rate Metrics
	fmt.Println("--- Example 3: Error Rate Metrics (per minute) ---")

	components := []struct {
		name       string
		ratePerMin float64
	}{
		{"llm-service", 2.5},
		{"api-client", 1.2},
		{"tool-executor", 3.8},
		{"gateway", 0.5},
	}

	for _, comp := range components {
		metrics.RecordErrorRate(ctx, comp.name, comp.ratePerMin)
		status := "🟢 Normal"
		if comp.ratePerMin > 5 {
			status = "🔴 Critical"
		} else if comp.ratePerMin > 2 {
			status = "🟡 Warning"
		}
		fmt.Printf("  %s: %.1f errors/min %s\n", comp.name, comp.ratePerMin, status)
	}
	fmt.Println()

	// Example 4: Health Status Monitoring
	fmt.Println("--- Example 4: Health Status Monitoring ---")

	healthStatuses := []struct {
		component string
		status    int64
		label     string
	}{
		{"llm-service", 2, "🟢 HEALTHY"},
		{"cache", 1, "🟡 DEGRADED"},
		{"database", 2, "🟢 HEALTHY"},
		{"external-api", 0, "🔴 UNHEALTHY"},
	}

	for _, hs := range healthStatuses {
		metrics.RecordHealthStatus(ctx, hs.component, hs.status)
		fmt.Printf("  %s: %s\n", hs.component, hs.label)
	}
	fmt.Println()

	// Example 5: Circuit Breaker State
	fmt.Println("--- Example 5: Circuit Breaker State Monitoring ---")

	circuitBreakerStates := []struct {
		component string
		state     int64
		label     string
		action    string
	}{
		{"api-client", 2, "🟢 CLOSED", "Normal operation, requests flowing"},
		{"external-service", 1, "🟡 HALF_OPEN", "Testing recovery, limited requests"},
		{"failing-dependency", 0, "🔴 OPEN", "Circuit broken, using fallback"},
	}

	for _, cbs := range circuitBreakerStates {
		metrics.RecordCircuitBreakerState(ctx, cbs.component, cbs.state)
		fmt.Printf("  %s: %s → %s\n", cbs.component, cbs.label, cbs.action)
	}
	fmt.Println()

	// Example 6: Real-Time Monitoring Scenario
	fmt.Println("--- Example 6: Real-Time Monitoring Scenario ---")
	fmt.Println("Simulating service degradation and recovery:")

	// Simulate LLM service degradation
	fmt.Println("T=0s: LLM service running normally")
	metrics.RecordHealthStatus(ctx, "llm-service", 2)
	metrics.RecordCircuitBreakerState(ctx, "llm-service", 2)

	time.Sleep(1 * time.Second)

	fmt.Println("T=1s: LLM service experiencing high load")
	metrics.RecordHealthStatus(ctx, "llm-service", 1)
	metrics.RecordErrorRate(ctx, "llm-service", 8.5)
	for i := 0; i < 5; i++ {
		err := errors.New(errors.CodeLLMError, "overloaded", nil).WithRecoverable(true)
		metrics.RecordErrorMetric(ctx, err, "llm-service")
	}

	time.Sleep(1 * time.Second)

	fmt.Println("T=2s: Circuit breaker opens, fallback activated")
	metrics.RecordHealthStatus(ctx, "llm-service", 0)
	metrics.RecordCircuitBreakerState(ctx, "llm-service", 0)
	metrics.RecordErrorRate(ctx, "llm-service", 15.2)

	time.Sleep(1 * time.Second)

	fmt.Println("T=3s: Recovery underway, circuit breaker half-open")
	metrics.RecordCircuitBreakerState(ctx, "llm-service", 1)

	time.Sleep(1 * time.Second)

	fmt.Println("T=4s: Service recovered, circuit breaker closed")
	metrics.RecordHealthStatus(ctx, "llm-service", 2)
	metrics.RecordCircuitBreakerState(ctx, "llm-service", 2)
	metrics.RecordErrorRate(ctx, "llm-service", 1.2)
	for i := 0; i < 4; i++ {
		metrics.RecordRecovery(ctx, errors.CodeLLMError)
	}
	fmt.Println()

	// Example 7: Alert Thresholds
	fmt.Println("--- Example 7: Alert Threshold Analysis ---")

	alertThresholds := []struct {
		name      string
		metric    string
		threshold string
		severity  string
		action    string
	}{
		{
			"HighErrorRate",
			"kairos.errors.total",
			"> 10/sec",
			"🔴 CRITICAL",
			"Page on-call, check service logs",
		},
		{
			"LowRecoveryRate",
			"recovery / total",
			"< 80%",
			"🟡 WARNING",
			"Review retry/fallback configurations",
		},
		{
			"CircuitBreakerOpen",
			"kairos.circuitbreaker.state",
			"== 0 (OPEN)",
			"🔴 CRITICAL",
			"Investigate component, check dependencies",
		},
		{
			"ComponentUnhealthy",
			"kairos.health.status",
			"== 0 (UNHEALTHY)",
			"🔴 CRITICAL",
			"Immediate investigation, possible failover",
		},
		{
			"NonRecoverableErrors",
			"kairos.errors.total{recoverable=false}",
			"> 1/sec",
			"🔴 CRITICAL",
			"Check for bugs or config issues",
		},
	}

	for _, alert := range alertThresholds {
		fmt.Printf("Alert: %s\n", alert.name)
		fmt.Printf("  Metric: %s\n", alert.metric)
		fmt.Printf("  Threshold: %s\n", alert.threshold)
		fmt.Printf("  Severity: %s\n", alert.severity)
		fmt.Printf("  Action: %s\n\n", alert.action)
	}

	// Example 8: Integration with Resilience Patterns
	fmt.Println("--- Example 8: Observability + Resilience Integration ---")
	fmt.Println("Demonstrating error metrics with retry/fallback patterns:")

	retryConfig := resilience.DefaultRetryConfig().
		WithMaxAttempts(3).
		WithInitialDelay(100 * time.Millisecond).
		WithMaxDelay(1 * time.Second)

	callAttempt := 0
	operation := func() error {
		callAttempt++
		if callAttempt < 3 {
			err := errors.New(errors.CodeTimeout, "operation timeout", nil).WithRecoverable(true)
			metrics.RecordErrorMetric(ctx, err, "operation")
			return err
		}
		metrics.RecordRecovery(ctx, errors.CodeTimeout)
		return nil
	}

	fmt.Printf("Executing operation with retry pattern:\n")
	err = retryConfig.Do(ctx, operation)
	if err != nil {
		fmt.Printf("  Failed after %d attempts\n", callAttempt)
	} else {
		fmt.Printf("  Succeeded on attempt %d\n", callAttempt)
	}
	fmt.Println()

	fmt.Println("=== Phase 3 Examples Completed ===")
	fmt.Println("\nMetrics exported to telemetry backend.")
	fmt.Println("Dashboard queries available in docs/internal/observability-dashboards.go")
	fmt.Println("Alert rules available in docs/internal/observability-dashboards.go")
}
