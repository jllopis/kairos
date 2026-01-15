// SPDX-License-Identifier: Apache-2.0
// Package resilience provides retry and circuit breaker patterns for Kairos.
// See docs/ERROR_HANDLING.md for strategy and examples.
package resilience

import (
	"context"

	"github.com/jllopis/kairos/pkg/errors"
)

// FallbackStrategy defines a fallback behavior when primary operation fails.
type FallbackStrategy interface {
	// Execute runs the fallback operation.
	Execute(ctx context.Context, primaryErr error) (interface{}, error)
}

// FallbackFunc wraps a function as a FallbackStrategy.
type FallbackFunc func(ctx context.Context, primaryErr error) (interface{}, error)

// Execute implements FallbackStrategy.
func (f FallbackFunc) Execute(ctx context.Context, err error) (interface{}, error) {
	return f(ctx, err)
}

// StaticFallback returns a static value on failure.
type StaticFallback struct {
	Value interface{}
}

// Execute implements FallbackStrategy.
func (s *StaticFallback) Execute(ctx context.Context, primaryErr error) (interface{}, error) {
	return s.Value, nil
}

// ErrorFallback returns an error with additional context.
type ErrorFallback struct {
	Message string
}

// Execute implements FallbackStrategy.
func (e *ErrorFallback) Execute(ctx context.Context, primaryErr error) (interface{}, error) {
	return nil, errors.New(errors.CodeInternal, e.Message, primaryErr).
		WithContext("fallback", "error").
		WithRecoverable(false)
}

// CachedFallback returns the last known good value on failure.
type CachedFallback struct {
	Cache interface{}
}

// Execute implements FallbackStrategy.
func (c *CachedFallback) Execute(ctx context.Context, primaryErr error) (interface{}, error) {
	if c.Cache == nil {
		return nil, errors.New(errors.CodeInternal, "no cached value available", primaryErr).
			WithContext("fallback", "cache").
			WithRecoverable(false)
	}
	return c.Cache, nil
}

// ChainedFallback tries multiple fallbacks in sequence.
type ChainedFallback struct {
	Fallbacks []FallbackStrategy
}

// Execute implements FallbackStrategy.
func (c *ChainedFallback) Execute(ctx context.Context, primaryErr error) (interface{}, error) {
	var lastErr error = primaryErr

	for _, fallback := range c.Fallbacks {
		value, err := fallback.Execute(ctx, lastErr)
		if err == nil {
			return value, nil
		}
		lastErr = err
	}

	return nil, lastErr
}

// WithFallback executes fn, and on error, uses the fallback strategy.
func WithFallback(ctx context.Context, fn func() (interface{}, error), fallback FallbackStrategy) (interface{}, error) {
	value, err := fn()
	if err == nil {
		return value, nil
	}

	return fallback.Execute(ctx, err)
}

// GracefulDegradation represents a service in degraded state.
type GracefulDegradation struct {
	Primary   func() (interface{}, error)
	Fallback  FallbackStrategy
	LogError  func(err error)
	MaxErrors int
	ErrorCount int
}

// Execute runs with fallback on error, tracking error count.
func (g *GracefulDegradation) Execute(ctx context.Context) (interface{}, error) {
	value, err := g.Primary()
	if err == nil {
		g.ErrorCount = 0 // Reset on success
		return value, nil
	}

	// Increment error count
	g.ErrorCount++
	if g.LogError != nil {
		g.LogError(err)
	}

	// If error threshold exceeded, use fallback
	if g.ErrorCount >= g.MaxErrors {
		return g.Fallback.Execute(ctx, err)
	}

	// Still propagate error, but caller can decide to retry or degrade
	return nil, err
}

// IsOperational returns true if the service is still operating normally.
func (g *GracefulDegradation) IsOperational() bool {
	return g.ErrorCount < g.MaxErrors
}

// Status returns the current operation status.
func (g *GracefulDegradation) Status() string {
	if g.IsOperational() {
		return "operational"
	}
	return "degraded"
}
