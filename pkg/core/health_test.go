// SPDX-License-Identifier: Apache-2.0
// Package core provides core interfaces for Kairos.
// See docs/ERROR_HANDLING.md for health check integration.
package core

import (
	"context"
	"testing"
	"time"
)

func TestHealthStatusConstants(t *testing.T) {
	tests := []struct {
		status HealthStatus
		name   string
	}{
		{HealthHealthy, "HEALTHY"},
		{HealthDegraded, "DEGRADED"},
		{HealthUnhealthy, "UNHEALTHY"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.name {
				t.Errorf("expected %q, got %q", tt.name, string(tt.status))
			}
		})
	}
}

func TestSimpleHealthChecker(t *testing.T) {
	tests := []struct {
		name   string
		status HealthStatus
	}{
		{"healthy", HealthHealthy},
		{"degraded", HealthDegraded},
		{"unhealthy", HealthUnhealthy},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewSimpleHealthChecker(tt.status, "test message")
			result := checker.Check(context.Background())

			if result.Status != tt.status {
				t.Errorf("expected %v, got %v", tt.status, result.Status)
			}
			if result.Message != "test message" {
				t.Errorf("expected message 'test message', got %q", result.Message)
			}
			if result.LastCheck.IsZero() {
				t.Errorf("expected LastCheck to be set")
			}
		})
	}
}

func TestFunctionHealthChecker(t *testing.T) {
	callCount := 0
	checker := NewFunctionHealthChecker(func(ctx context.Context) HealthResult {
		callCount++
		return HealthResult{
			Status:  HealthHealthy,
			Message: "ok",
		}
	})

	result := checker.Check(context.Background())
	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}
	if result.Status != HealthHealthy {
		t.Errorf("expected Healthy")
	}
	if result.LastCheck.IsZero() {
		t.Errorf("expected LastCheck to be set by wrapper")
	}
}

func TestDefaultHealthCheckProvider(t *testing.T) {
	provider := NewDefaultHealthCheckProvider(10 * time.Second)

	provider.RegisterChecker("service1", NewSimpleHealthChecker(HealthHealthy, "ok"))
	provider.RegisterChecker("service2", NewSimpleHealthChecker(HealthDegraded, "slow"))
	provider.RegisterChecker("service3", NewSimpleHealthChecker(HealthUnhealthy, "down"))

	results, overallStatus := provider.CheckAll(context.Background())

	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}

	// Overall status should be Unhealthy if any component is Unhealthy
	if overallStatus != HealthUnhealthy {
		t.Errorf("expected Unhealthy overall, got %v", overallStatus)
	}
}

func TestDefaultHealthCheckProviderDegraded(t *testing.T) {
	provider := NewDefaultHealthCheckProvider(10 * time.Second)

	provider.RegisterChecker("service1", NewSimpleHealthChecker(HealthHealthy, "ok"))
	provider.RegisterChecker("service2", NewSimpleHealthChecker(HealthDegraded, "slow"))

	results, overallStatus := provider.CheckAll(context.Background())

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	// Overall status should be Degraded if no Unhealthy but some Degraded
	if overallStatus != HealthDegraded {
		t.Errorf("expected Degraded overall, got %v", overallStatus)
	}
}

func TestDefaultHealthCheckProviderHealthy(t *testing.T) {
	provider := NewDefaultHealthCheckProvider(10 * time.Second)

	provider.RegisterChecker("service1", NewSimpleHealthChecker(HealthHealthy, "ok"))
	provider.RegisterChecker("service2", NewSimpleHealthChecker(HealthHealthy, "ok"))

	results, overallStatus := provider.CheckAll(context.Background())

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	// Overall status should be Healthy if all are Healthy
	if overallStatus != HealthHealthy {
		t.Errorf("expected Healthy overall, got %v", overallStatus)
	}
}

func TestCheckSpecific(t *testing.T) {
	provider := NewDefaultHealthCheckProvider(10 * time.Second)
	provider.RegisterChecker("service", NewSimpleHealthChecker(HealthHealthy, "ok"))

	result, err := provider.Check(context.Background(), "service")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result.Status != HealthHealthy {
		t.Errorf("expected Healthy")
	}
}

func TestCheckSpecificNotFound(t *testing.T) {
	provider := NewDefaultHealthCheckProvider(10 * time.Second)

	_, err := provider.Check(context.Background(), "nonexistent")
	if err == nil {
		t.Errorf("expected error for nonexistent checker")
	}
}

func TestCheckWithContext(t *testing.T) {
	provider := NewDefaultHealthCheckProvider(10 * time.Second)

	// Checker that respects context timeout
	checker := NewFunctionHealthChecker(func(ctx context.Context) HealthResult {
		select {
		case <-ctx.Done():
			return HealthResult{
				Status:  HealthUnhealthy,
				Message: "context timeout",
			}
		case <-time.After(100 * time.Millisecond):
			return HealthResult{
				Status:  HealthHealthy,
				Message: "ok",
			}
		}
	})

	provider.RegisterChecker("slow_service", checker)

	// With timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	result, _ := provider.Check(ctx, "slow_service")
	if result.Status != HealthUnhealthy {
		t.Errorf("expected Unhealthy due to timeout")
	}
}

func TestRecoverableInterface(t *testing.T) {
	// Test that DefaultHealthCheckProvider properly manages checkers
	provider := NewDefaultHealthCheckProvider(10 * time.Second)
	checker := NewSimpleHealthChecker(HealthHealthy, "test")
	provider.RegisterChecker("test", checker)

	result, err := provider.Check(context.Background(), "test")
	if err != nil {
		t.Errorf("Check failed: %v", err)
	}
	if result.Status != HealthHealthy {
		t.Errorf("expected HealthHealthy, got %v", result.Status)
	}
}
