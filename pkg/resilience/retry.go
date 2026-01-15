// SPDX-License-Identifier: Apache-2.0
// Package resilience provides retry and circuit breaker patterns for Kairos.
// See docs/ERROR_HANDLING.md for strategy and examples.
package resilience

import (
	"context"
	"math"
	"math/rand"
	"time"

	"github.com/jllopis/kairos/pkg/errors"
)

// RetryConfig controls retry behavior with exponential backoff.
type RetryConfig struct {
	// MaxAttempts is the maximum number of attempts (must be >= 1).
	MaxAttempts int

	// InitialDelay is the initial backoff delay.
	InitialDelay time.Duration

	// MaxDelay caps the exponential backoff delay.
	MaxDelay time.Duration

	// Multiplier for exponential backoff (default 2.0).
	Multiplier float64

	// IsRecoverable determines if an error should be retried.
	// If nil, all errors are considered recoverable.
	IsRecoverable func(error) bool

	// Jitter adds randomness to backoff to prevent thundering herd.
	// Value between 0 and 1; 0.1 means ±10% jitter.
	Jitter float64
}

// DefaultRetryConfig returns a sensible default retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:   3,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      10 * time.Second,
		Multiplier:    2.0,
		Jitter:        0.1,
		IsRecoverable: isRecoverableDefault,
	}
}

// WithMaxAttempts returns a new config with MaxAttempts set.
func (rc RetryConfig) WithMaxAttempts(max int) RetryConfig {
	rc.MaxAttempts = max
	return rc
}

// WithInitialDelay returns a new config with InitialDelay set.
func (rc RetryConfig) WithInitialDelay(d time.Duration) RetryConfig {
	rc.InitialDelay = d
	return rc
}

// WithMaxDelay returns a new config with MaxDelay set.
func (rc RetryConfig) WithMaxDelay(d time.Duration) RetryConfig {
	rc.MaxDelay = d
	return rc
}

// WithIsRecoverable returns a new config with IsRecoverable set.
func (rc RetryConfig) WithIsRecoverable(fn func(error) bool) RetryConfig {
	rc.IsRecoverable = fn
	return rc
}

// Do executes fn with retry logic, returning the last error if all attempts fail.
func (rc RetryConfig) Do(ctx context.Context, fn func() error) error {
	if rc.MaxAttempts < 1 {
		rc.MaxAttempts = 1
	}
	if rc.IsRecoverable == nil {
		rc.IsRecoverable = isRecoverableDefault
	}

	var lastErr error
	for attempt := 0; attempt < rc.MaxAttempts; attempt++ {
		// Apply backoff before attempt (skip first attempt)
		if attempt > 0 {
			delay := calculateBackoff(attempt, rc)
			select {
			case <-ctx.Done():
				return errors.New(errors.CodeContextLost, "context canceled during retry", ctx.Err()).
					WithContext("attempt", attempt).
					WithContext("max_attempts", rc.MaxAttempts)
			case <-time.After(delay):
				// Proceed to retry
			}
		}

		// Execute function
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is recoverable
		if !rc.IsRecoverable(err) {
			return err
		}
	}

	return lastErr
}

// DoWithResult executes fn with retry logic, returning both result and error.
func (rc RetryConfig) DoWithResult(ctx context.Context, fn func() (interface{}, error)) (interface{}, error) {
	var result interface{}
	err := rc.Do(ctx, func() error {
		var fnErr error
		result, fnErr = fn()
		return fnErr
	})
	return result, err
}

// calculateBackoff computes exponential backoff delay with jitter.
func calculateBackoff(attempt int, rc RetryConfig) time.Duration {
	if rc.Multiplier == 0 {
		rc.Multiplier = 2.0
	}

	// Exponential backoff: initialDelay * multiplier^attempt
	exponentialDelay := time.Duration(float64(rc.InitialDelay) * math.Pow(rc.Multiplier, float64(attempt)))

	// Cap at MaxDelay
	if exponentialDelay > rc.MaxDelay {
		exponentialDelay = rc.MaxDelay
	}

	// Apply jitter
	if rc.Jitter > 0 {
		jitterAmount := exponentialDelay.Seconds() * rc.Jitter
		jitterRange := 2 * jitterAmount * (rand.Float64() - 0.5)
		exponentialDelay = time.Duration(float64(exponentialDelay) + jitterRange*1e9)
		if exponentialDelay < 0 {
			exponentialDelay = 0
		}
	}

	return exponentialDelay
}

// isRecoverableDefault considers errors recoverable based on type.
func isRecoverableDefault(err error) bool {
	if err == nil {
		return false
	}

	// Check if it's a KairosError with explicit recoverable flag
	if ke, ok := err.(*errors.KairosError); ok {
		return ke.Recoverable
	}

	// Default: all generic errors are considered recoverable for backward compatibility
	// Callers can override with their own IsRecoverable function for finer control
	return true
}
