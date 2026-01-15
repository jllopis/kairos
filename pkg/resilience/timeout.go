// SPDX-License-Identifier: Apache-2.0
// Package resilience provides retry and circuit breaker patterns for Kairos.
// See docs/ERROR_HANDLING.md for strategy and examples.
package resilience

import (
	"context"
	"time"

	"github.com/jllopis/kairos/pkg/errors"
)

// TimeoutConfig controls timeout behavior.
type TimeoutConfig struct {
	// Duration is the maximum time allowed for the operation.
	Duration time.Duration

	// ErrorOnTimeout determines if an error should be returned on timeout.
	// If false, a default value or nil is used.
	ErrorOnTimeout bool
}

// WithTimeout executes fn with a timeout boundary.
// Returns errors.CodeTimeout if the deadline is exceeded.
func WithTimeout(ctx context.Context, config TimeoutConfig, fn func() error) error {
	if config.Duration == 0 {
		return fn()
	}

	ctx, cancel := context.WithTimeout(ctx, config.Duration)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- fn()
	}()

	select {
	case <-ctx.Done():
		return errors.New(errors.CodeTimeout, "operation exceeded timeout", ctx.Err()).
			WithContext("timeout", config.Duration.String()).
			WithRecoverable(true)
	case err := <-done:
		return err
	}
}

// WithTimeoutResult executes fn with a timeout boundary, returning both result and error.
func WithTimeoutResult(ctx context.Context, config TimeoutConfig, fn func() (interface{}, error)) (interface{}, error) {
	if config.Duration == 0 {
		return fn()
	}

	ctx, cancel := context.WithTimeout(ctx, config.Duration)
	defer cancel()

	type result struct {
		value interface{}
		err   error
	}

	done := make(chan result, 1)
	go func() {
		value, err := fn()
		done <- result{value, err}
	}()

	select {
	case <-ctx.Done():
		return nil, errors.New(errors.CodeTimeout, "operation exceeded timeout", ctx.Err()).
			WithContext("timeout", config.Duration.String()).
			WithRecoverable(true)
	case res := <-done:
		return res.value, res.err
	}
}
