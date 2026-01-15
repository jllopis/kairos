// SPDX-License-Identifier: Apache-2.0
// Package core provides core interfaces for Kairos.
// See docs/ERROR_HANDLING.md for health check integration.
package core

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// DefaultHealthCheckProvider implements HealthCheckProvider.
type DefaultHealthCheckProvider struct {
	checkers map[string]HealthChecker
	mu       sync.RWMutex
	cache    map[string]HealthResult
	cacheTTL time.Duration
}

// NewDefaultHealthCheckProvider creates a new health check provider.
func NewDefaultHealthCheckProvider(cacheTTL time.Duration) *DefaultHealthCheckProvider {
	if cacheTTL == 0 {
		cacheTTL = 10 * time.Second
	}
	return &DefaultHealthCheckProvider{
		checkers: make(map[string]HealthChecker),
		cache:    make(map[string]HealthResult),
		cacheTTL: cacheTTL,
	}
}

// RegisterChecker registers a health checker for a component.
func (p *DefaultHealthCheckProvider) RegisterChecker(name string, checker HealthChecker) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.checkers[name] = checker
}

// Check checks the health of a specific component.
func (p *DefaultHealthCheckProvider) Check(ctx context.Context, name string) (HealthResult, error) {
	p.mu.RLock()
	checker, exists := p.checkers[name]
	p.mu.RUnlock()

	if !exists {
		return HealthResult{}, fmt.Errorf("checker not registered: %s", name)
	}

	return checker.Check(ctx), nil
}

// CheckAll checks the health of all registered components.
// Returns individual results and overall status (Healthy only if all Healthy).
func (p *DefaultHealthCheckProvider) CheckAll(ctx context.Context) ([]HealthResult, HealthStatus) {
	p.mu.RLock()
	checkerCount := len(p.checkers)
	p.mu.RUnlock()

	results := make([]HealthResult, 0, checkerCount)
	healthyCount := 0
	degradedCount := 0
	unhealthyCount := 0

	// Check all components
	for name, checker := range p.getAllCheckers() {
		result := checker.Check(ctx)
		result.Component = name
		results = append(results, result)

		switch result.Status {
		case HealthHealthy:
			healthyCount++
		case HealthDegraded:
			degradedCount++
		case HealthUnhealthy:
			unhealthyCount++
		}
	}

	// Determine overall status
	overallStatus := HealthHealthy
	if unhealthyCount > 0 {
		overallStatus = HealthUnhealthy
	} else if degradedCount > 0 {
		overallStatus = HealthDegraded
	}

	return results, overallStatus
}

// getAllCheckers returns a snapshot of all checkers.
func (p *DefaultHealthCheckProvider) getAllCheckers() map[string]HealthChecker {
	p.mu.RLock()
	defer p.mu.RUnlock()

	checkers := make(map[string]HealthChecker, len(p.checkers))
	for name, checker := range p.checkers {
		checkers[name] = checker
	}
	return checkers
}

// SimpleHealthChecker is a basic health checker that returns a constant status.
// Useful for testing or components with static health.
type SimpleHealthChecker struct {
	status  HealthStatus
	message string
}

// NewSimpleHealthChecker creates a new simple health checker.
func NewSimpleHealthChecker(status HealthStatus, message string) *SimpleHealthChecker {
	return &SimpleHealthChecker{
		status:  status,
		message: message,
	}
}

// Check returns the constant health status.
func (s *SimpleHealthChecker) Check(ctx context.Context) HealthResult {
	return HealthResult{
		Status:    s.status,
		Message:   s.message,
		LastCheck: time.Now(),
	}
}

// FunctionHealthChecker wraps a function as a health checker.
type FunctionHealthChecker struct {
	fn func(ctx context.Context) HealthResult
}

// NewFunctionHealthChecker creates a health checker from a function.
func NewFunctionHealthChecker(fn func(ctx context.Context) HealthResult) *FunctionHealthChecker {
	return &FunctionHealthChecker{fn: fn}
}

// Check calls the underlying function.
func (f *FunctionHealthChecker) Check(ctx context.Context) HealthResult {
	result := f.fn(ctx)
	if result.LastCheck.IsZero() {
		result.LastCheck = time.Now()
	}
	return result
}
