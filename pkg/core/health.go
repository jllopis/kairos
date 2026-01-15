// SPDX-License-Identifier: Apache-2.0
// Package core provides core interfaces for Kairos.
// See docs/ERROR_HANDLING.md for health check integration.
package core

import (
	"context"
	"time"
)

// HealthStatus represents the health state of a component.
type HealthStatus string

const (
	// HealthHealthy indicates the component is fully operational.
	HealthHealthy HealthStatus = "HEALTHY"

	// HealthDegraded indicates the component is operational but with reduced capacity.
	HealthDegraded HealthStatus = "DEGRADED"

	// HealthUnhealthy indicates the component is not operational.
	HealthUnhealthy HealthStatus = "UNHEALTHY"
)

// HealthResult represents the result of a health check.
type HealthResult struct {
	Status    HealthStatus
	Component string
	Message   string
	LastCheck time.Time
	Error     error
}

// HealthChecker checks the health of a component.
type HealthChecker interface {
	// Check returns the current health status of the component.
	// The context can be used to implement timeouts.
	Check(ctx context.Context) HealthResult
}

// HealthCheckProvider provides health check results for multiple components.
type HealthCheckProvider interface {
	// RegisterChecker registers a health checker for a component.
	RegisterChecker(name string, checker HealthChecker)

	// CheckAll checks the health of all registered components.
	// Returns individual results and overall status.
	CheckAll(ctx context.Context) ([]HealthResult, HealthStatus)

	// Check checks the health of a specific component.
	Check(ctx context.Context, name string) (HealthResult, error)
}

// Recoverable determines if an operation should be retried based on the component's health.
// A recoverable operation is one that can succeed after transient failures or degradation.
type Recoverable interface {
	// IsRecoverable returns true if the operation can be retried.
	IsRecoverable(health HealthStatus) bool
}
