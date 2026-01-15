// SPDX-License-Identifier: Apache-2.0
// Package resilience provides retry and circuit breaker patterns for Kairos.
// See docs/ERROR_HANDLING.md for strategy and examples.
package resilience

import (
	"context"
	"sync"
	"time"

	"github.com/jllopis/kairos/pkg/errors"
)

// CircuitBreakerState represents the state of a circuit breaker.
type CircuitBreakerState string

const (
	// StateClosed means the circuit breaker is working normally.
	StateClosed CircuitBreakerState = "closed"

	// StateOpen means the circuit breaker is blocking calls.
	StateOpen CircuitBreakerState = "open"

	// StateHalfOpen means the circuit breaker is testing if service recovered.
	StateHalfOpen CircuitBreakerState = "half-open"
)

// CircuitBreakerConfig configures a circuit breaker.
type CircuitBreakerConfig struct {
	// FailureThreshold is the number of failures before opening the circuit.
	FailureThreshold int

	// SuccessThreshold is the number of successes in half-open before closing.
	SuccessThreshold int

	// Timeout is how long to wait before trying half-open state.
	Timeout time.Duration

	// Name is the circuit breaker identifier for logging/metrics.
	Name string
}

// CircuitBreaker prevents cascading failures using the circuit breaker pattern.
type CircuitBreaker struct {
	config       CircuitBreakerConfig
	state        CircuitBreakerState
	failures     int
	successes    int
	lastFailTime time.Time
	mu           sync.RWMutex
}

// NewCircuitBreaker creates a new circuit breaker with the given config.
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	if config.FailureThreshold < 1 {
		config.FailureThreshold = 5
	}
	if config.SuccessThreshold < 1 {
		config.SuccessThreshold = 2
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.Name == "" {
		config.Name = "circuit_breaker"
	}

	return &CircuitBreaker{
		config: config,
		state:  StateClosed,
	}
}

// Call executes fn if the circuit breaker allows, tracking success/failure.
// Returns errors.CodeInternal if the circuit is open.
func (cb *CircuitBreaker) Call(ctx context.Context, fn func() error) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// Check state and potentially transition
	cb.checkState()

	// If open, reject immediately
	if cb.state == StateOpen {
		return errors.New(errors.CodeInternal, "circuit breaker open", nil).
			WithContext("breaker", cb.config.Name).
			WithRecoverable(true)
	}

	// Execute function
	err := fn()

	// Update state based on result
	if err != nil {
		cb.failures++
		cb.lastFailTime = time.Now()

		// Transition to open if threshold reached
		if cb.failures >= cb.config.FailureThreshold && cb.state == StateClosed {
			cb.state = StateOpen
			cb.failures = 0
			cb.successes = 0
		}
	} else {
		// Success
		if cb.state == StateHalfOpen {
			cb.successes++
			if cb.successes >= cb.config.SuccessThreshold {
				// Recover to closed state
				cb.state = StateClosed
				cb.failures = 0
				cb.successes = 0
			}
		} else if cb.state == StateClosed {
			// Reset failure count on success in closed state
			cb.failures = 0
		}
	}

	return err
}

// checkState transitions the circuit breaker state if appropriate.
// Must be called under lock.
func (cb *CircuitBreaker) checkState() {
	if cb.state == StateOpen {
		// Check if we should try half-open
		if time.Since(cb.lastFailTime) > cb.config.Timeout {
			cb.state = StateHalfOpen
			cb.successes = 0
			cb.failures = 0
		}
	}
}

// State returns the current circuit breaker state.
func (cb *CircuitBreaker) State() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Reset manually resets the circuit breaker to closed state.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = StateClosed
	cb.failures = 0
	cb.successes = 0
}

// Open manually forces the circuit breaker to open state.
func (cb *CircuitBreaker) Open() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = StateOpen
	cb.lastFailTime = time.Now()
}
