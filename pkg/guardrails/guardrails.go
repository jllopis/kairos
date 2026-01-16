// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

// Package guardrails provides content safety filters for AI agent inputs and outputs.
//
// Guardrails run at two points in the agent loop:
//   - Input: Before user message reaches the LLM (prompt injection, forbidden topics)
//   - Output: Before LLM response is returned to user (PII, harmful content)
//
// Unlike governance policies (which gate actions like tool calls), guardrails
// inspect and filter the actual content of messages.
//
// Example usage:
//
//	guard := guardrails.New(
//	    guardrails.WithPromptInjectionDetector(),
//	    guardrails.WithPIIFilter(guardrails.PIIFilterMask),
//	    guardrails.WithContentFilter("profanity", "violence"),
//	)
//
//	// In agent loop
//	result := guard.CheckInput(ctx, userMessage)
//	if result.Blocked {
//	    return result.Reason
//	}
//
//	// After LLM response
//	output := guard.FilterOutput(ctx, llmResponse)
package guardrails

import (
	"context"
	"sync"
)

// CheckResult represents the outcome of a guardrail check.
type CheckResult struct {
	// Blocked indicates the content should not proceed.
	Blocked bool

	// Reason explains why content was blocked (empty if not blocked).
	Reason string

	// GuardrailID identifies which guardrail triggered the block.
	GuardrailID string

	// Confidence is the detection confidence (0.0-1.0) for ML-based detectors.
	Confidence float64

	// Metadata contains additional context from the check.
	Metadata map[string]any
}

// FilterResult represents the outcome of output filtering.
type FilterResult struct {
	// Content is the (potentially modified) output content.
	Content string

	// Modified indicates if the content was changed.
	Modified bool

	// Redactions lists what was removed or masked.
	Redactions []Redaction
}

// Redaction describes a single content modification.
type Redaction struct {
	// Type categorizes the redaction (e.g., "pii:email", "profanity").
	Type string

	// Original is the redacted text (may be empty for privacy).
	Original string

	// Replacement is what replaced the original.
	Replacement string

	// Position is the character offset in original content.
	Position int
}

// InputChecker validates content before it reaches the LLM.
type InputChecker interface {
	// CheckInput examines user input for policy violations.
	// Returns a CheckResult indicating if the input should be blocked.
	CheckInput(ctx context.Context, input string) CheckResult

	// ID returns a unique identifier for this checker.
	ID() string
}

// OutputFilter processes LLM output before returning to user.
type OutputFilter interface {
	// FilterOutput processes and potentially modifies LLM output.
	// Returns filtered content and metadata about changes.
	FilterOutput(ctx context.Context, output string) FilterResult

	// ID returns a unique identifier for this filter.
	ID() string
}

// Guardrails orchestrates multiple input checkers and output filters.
type Guardrails struct {
	mu             sync.RWMutex
	inputCheckers  []InputChecker
	outputFilters  []OutputFilter
	failOpen       bool // If true, errors in checks allow content through
	collectMetrics bool
}

// Option configures the Guardrails instance.
type Option func(*Guardrails)

// New creates a new Guardrails instance with the given options.
func New(opts ...Option) *Guardrails {
	g := &Guardrails{
		inputCheckers:  make([]InputChecker, 0),
		outputFilters:  make([]OutputFilter, 0),
		failOpen:       false, // Secure by default
		collectMetrics: true,
	}

	for _, opt := range opts {
		opt(g)
	}

	return g
}

// WithInputChecker adds an input checker to the guardrails.
func WithInputChecker(checker InputChecker) Option {
	return func(g *Guardrails) {
		g.inputCheckers = append(g.inputCheckers, checker)
	}
}

// WithOutputFilter adds an output filter to the guardrails.
func WithOutputFilter(filter OutputFilter) Option {
	return func(g *Guardrails) {
		g.outputFilters = append(g.outputFilters, filter)
	}
}

// WithFailOpen configures guardrails to allow content through on errors.
// Default is fail-closed (secure by default).
func WithFailOpen(failOpen bool) Option {
	return func(g *Guardrails) {
		g.failOpen = failOpen
	}
}

// WithMetrics enables or disables metrics collection.
func WithMetrics(enabled bool) Option {
	return func(g *Guardrails) {
		g.collectMetrics = enabled
	}
}

// CheckInput runs all input checkers and returns the first blocking result.
// If no checker blocks, returns an empty CheckResult with Blocked=false.
func (g *Guardrails) CheckInput(ctx context.Context, input string) CheckResult {
	g.mu.RLock()
	checkers := g.inputCheckers
	g.mu.RUnlock()

	for _, checker := range checkers {
		select {
		case <-ctx.Done():
			if g.failOpen {
				return CheckResult{Blocked: false}
			}
			return CheckResult{
				Blocked:     true,
				Reason:      "guardrail check cancelled",
				GuardrailID: "system",
			}
		default:
		}

		result := checker.CheckInput(ctx, input)
		if result.Blocked {
			result.GuardrailID = checker.ID()
			return result
		}
	}

	return CheckResult{Blocked: false}
}

// FilterOutput runs all output filters in sequence.
// Each filter receives the output of the previous one.
func (g *Guardrails) FilterOutput(ctx context.Context, output string) FilterResult {
	g.mu.RLock()
	filters := g.outputFilters
	g.mu.RUnlock()

	result := FilterResult{
		Content:    output,
		Modified:   false,
		Redactions: make([]Redaction, 0),
	}

	for _, filter := range filters {
		select {
		case <-ctx.Done():
			return result
		default:
		}

		filterResult := filter.FilterOutput(ctx, result.Content)
		if filterResult.Modified {
			result.Content = filterResult.Content
			result.Modified = true
			result.Redactions = append(result.Redactions, filterResult.Redactions...)
		}
	}

	return result
}

// AddInputChecker adds a checker at runtime.
func (g *Guardrails) AddInputChecker(checker InputChecker) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.inputCheckers = append(g.inputCheckers, checker)
}

// AddOutputFilter adds a filter at runtime.
func (g *Guardrails) AddOutputFilter(filter OutputFilter) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.outputFilters = append(g.outputFilters, filter)
}

// RemoveInputChecker removes a checker by ID.
func (g *Guardrails) RemoveInputChecker(id string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	for i, checker := range g.inputCheckers {
		if checker.ID() == id {
			g.inputCheckers = append(g.inputCheckers[:i], g.inputCheckers[i+1:]...)
			return true
		}
	}
	return false
}

// RemoveOutputFilter removes a filter by ID.
func (g *Guardrails) RemoveOutputFilter(id string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	for i, filter := range g.outputFilters {
		if filter.ID() == id {
			g.outputFilters = append(g.outputFilters[:i], g.outputFilters[i+1:]...)
			return true
		}
	}
	return false
}

// Stats returns current guardrails statistics.
func (g *Guardrails) Stats() Stats {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return Stats{
		InputCheckers: len(g.inputCheckers),
		OutputFilters: len(g.outputFilters),
		FailOpen:      g.failOpen,
	}
}

// Stats contains guardrails runtime statistics.
type Stats struct {
	InputCheckers int
	OutputFilters int
	FailOpen      bool
}
