// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

// Package testing provides utilities for testing Kairos agents and flows.
//
// This package includes:
//   - Scenario definitions for declarative agent testing
//   - Mock providers with scripted responses
//   - Assertion helpers for common validations
//   - Event collectors for verifying agent behavior
//
// Example usage:
//
//	scenario := testing.NewScenario("greeting test").
//	    WithInput("Hello").
//	    ExpectOutput(testing.Contains("Hello")).
//	    ExpectNoToolCalls()
//
//	result := scenario.Run(t, agent)
//	result.Assert(t)
package testing

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jllopis/kairos/pkg/core"
	"github.com/jllopis/kairos/pkg/llm"
)

// Scenario defines a test scenario for an agent interaction.
type Scenario struct {
	name           string
	description    string
	input          string
	context        context.Context
	timeout        time.Duration
	expectations   []Expectation
	setupFuncs     []func() error
	teardownFuncs  []func() error
}

// Expectation defines a condition to verify after running a scenario.
type Expectation interface {
	// Check verifies the expectation against the result.
	Check(result *ScenarioResult) error
	// Description returns a human-readable description of the expectation.
	Description() string
}

// ScenarioResult contains the outcome of running a scenario.
type ScenarioResult struct {
	Output     string
	Error      error
	Events     []core.Event
	ToolCalls  []ToolCallRecord
	Duration   time.Duration
	TokenUsage llm.Usage
}

// ToolCallRecord records a tool call made during the scenario.
type ToolCallRecord struct {
	Name      string
	Arguments map[string]any
	Result    string
	Error     error
	Duration  time.Duration
}

// NewScenario creates a new test scenario with the given name.
func NewScenario(name string) *Scenario {
	return &Scenario{
		name:         name,
		timeout:      30 * time.Second,
		context:      context.Background(),
		expectations: make([]Expectation, 0),
	}
}

// WithDescription adds a description to the scenario.
func (s *Scenario) WithDescription(desc string) *Scenario {
	s.description = desc
	return s
}

// WithInput sets the input prompt for the scenario.
func (s *Scenario) WithInput(input string) *Scenario {
	s.input = input
	return s
}

// WithContext sets the context for the scenario.
func (s *Scenario) WithContext(ctx context.Context) *Scenario {
	s.context = ctx
	return s
}

// WithTimeout sets the timeout for the scenario.
func (s *Scenario) WithTimeout(d time.Duration) *Scenario {
	s.timeout = d
	return s
}

// WithSetup adds a setup function to run before the scenario.
func (s *Scenario) WithSetup(fn func() error) *Scenario {
	s.setupFuncs = append(s.setupFuncs, fn)
	return s
}

// WithTeardown adds a teardown function to run after the scenario.
func (s *Scenario) WithTeardown(fn func() error) *Scenario {
	s.teardownFuncs = append(s.teardownFuncs, fn)
	return s
}

// Expect adds an expectation to the scenario.
func (s *Scenario) Expect(exp Expectation) *Scenario {
	s.expectations = append(s.expectations, exp)
	return s
}

// ExpectOutput adds an output expectation.
func (s *Scenario) ExpectOutput(matcher StringMatcher) *Scenario {
	return s.Expect(&outputExpectation{matcher: matcher})
}

// ExpectNoError expects no error from the agent.
func (s *Scenario) ExpectNoError() *Scenario {
	return s.Expect(&noErrorExpectation{})
}

// ExpectError expects an error matching the given pattern.
func (s *Scenario) ExpectError(matcher StringMatcher) *Scenario {
	return s.Expect(&errorExpectation{matcher: matcher})
}

// ExpectToolCall expects a specific tool to be called.
func (s *Scenario) ExpectToolCall(toolName string) *Scenario {
	return s.Expect(&toolCallExpectation{toolName: toolName})
}

// ExpectNoToolCalls expects no tool calls.
func (s *Scenario) ExpectNoToolCalls() *Scenario {
	return s.Expect(&noToolCallsExpectation{})
}

// ExpectEvent expects an event of the given type.
func (s *Scenario) ExpectEvent(eventType core.EventType) *Scenario {
	return s.Expect(&eventExpectation{eventType: eventType})
}

// ExpectMinDuration expects the scenario to take at least the given duration.
func (s *Scenario) ExpectMinDuration(d time.Duration) *Scenario {
	return s.Expect(&minDurationExpectation{min: d})
}

// ExpectMaxDuration expects the scenario to complete within the given duration.
func (s *Scenario) ExpectMaxDuration(d time.Duration) *Scenario {
	return s.Expect(&maxDurationExpectation{max: d})
}

// AgentRunner is the interface for running agent scenarios.
type AgentRunner interface {
	Run(ctx context.Context, input string) (string, error)
}

// Run executes the scenario against the given agent.
func (s *Scenario) Run(t *testing.T, agent AgentRunner) *ScenarioResult {
	t.Helper()

	// Run setup
	for _, setup := range s.setupFuncs {
		if err := setup(); err != nil {
			t.Fatalf("scenario %q setup failed: %v", s.name, err)
		}
	}

	// Ensure teardown runs
	defer func() {
		for _, teardown := range s.teardownFuncs {
			if err := teardown(); err != nil {
				t.Errorf("scenario %q teardown failed: %v", s.name, err)
			}
		}
	}()

	ctx, cancel := context.WithTimeout(s.context, s.timeout)
	defer cancel()

	start := time.Now()
	output, err := agent.Run(ctx, s.input)
	duration := time.Since(start)

	return &ScenarioResult{
		Output:   output,
		Error:    err,
		Duration: duration,
	}
}

// Assert checks all expectations and reports failures to the test.
func (r *ScenarioResult) Assert(t *testing.T, scenario *Scenario) {
	t.Helper()

	for _, exp := range scenario.expectations {
		if err := exp.Check(r); err != nil {
			t.Errorf("expectation %q failed: %v", exp.Description(), err)
		}
	}
}

// StringMatcher defines how to match strings in expectations.
type StringMatcher interface {
	Match(s string) bool
	Description() string
}

// Contains returns a matcher that checks if the string contains the substring.
func Contains(substr string) StringMatcher {
	return &containsMatcher{substr: substr}
}

// Equals returns a matcher that checks exact string equality.
func Equals(expected string) StringMatcher {
	return &equalsMatcher{expected: expected}
}

// Regex returns a matcher that checks against a regular expression.
func Regex(pattern string) StringMatcher {
	return &regexMatcher{pattern: pattern}
}

// HasPrefix returns a matcher that checks if the string has the given prefix.
func HasPrefix(prefix string) StringMatcher {
	return &prefixMatcher{prefix: prefix}
}

// HasSuffix returns a matcher that checks if the string has the given suffix.
func HasSuffix(suffix string) StringMatcher {
	return &suffixMatcher{suffix: suffix}
}

// containsMatcher checks if a string contains a substring.
type containsMatcher struct {
	substr string
}

func (m *containsMatcher) Match(s string) bool {
	return strings.Contains(s, m.substr)
}

func (m *containsMatcher) Description() string {
	return fmt.Sprintf("contains %q", m.substr)
}

// equalsMatcher checks exact string equality.
type equalsMatcher struct {
	expected string
}

func (m *equalsMatcher) Match(s string) bool {
	return s == m.expected
}

func (m *equalsMatcher) Description() string {
	return fmt.Sprintf("equals %q", m.expected)
}

// regexMatcher checks against a regular expression.
type regexMatcher struct {
	pattern string
}

func (m *regexMatcher) Match(s string) bool {
	re, err := regexp.Compile(m.pattern)
	if err != nil {
		return false
	}
	return re.MatchString(s)
}

func (m *regexMatcher) Description() string {
	return fmt.Sprintf("matches regex %q", m.pattern)
}

// prefixMatcher checks if a string has the given prefix.
type prefixMatcher struct {
	prefix string
}

func (m *prefixMatcher) Match(s string) bool {
	return strings.HasPrefix(s, m.prefix)
}

func (m *prefixMatcher) Description() string {
	return fmt.Sprintf("has prefix %q", m.prefix)
}

// suffixMatcher checks if a string has the given suffix.
type suffixMatcher struct {
	suffix string
}

func (m *suffixMatcher) Match(s string) bool {
	return strings.HasSuffix(s, m.suffix)
}

func (m *suffixMatcher) Description() string {
	return fmt.Sprintf("has suffix %q", m.suffix)
}

// Expectation implementations

type outputExpectation struct {
	matcher StringMatcher
}

func (e *outputExpectation) Check(r *ScenarioResult) error {
	if !e.matcher.Match(r.Output) {
		return fmt.Errorf("output %q does not match: %s", r.Output, e.matcher.Description())
	}
	return nil
}

func (e *outputExpectation) Description() string {
	return fmt.Sprintf("output %s", e.matcher.Description())
}

type noErrorExpectation struct{}

func (e *noErrorExpectation) Check(r *ScenarioResult) error {
	if r.Error != nil {
		return fmt.Errorf("expected no error, got: %v", r.Error)
	}
	return nil
}

func (e *noErrorExpectation) Description() string {
	return "no error"
}

type errorExpectation struct {
	matcher StringMatcher
}

func (e *errorExpectation) Check(r *ScenarioResult) error {
	if r.Error == nil {
		return fmt.Errorf("expected error matching %s, got nil", e.matcher.Description())
	}
	if !e.matcher.Match(r.Error.Error()) {
		return fmt.Errorf("error %q does not match: %s", r.Error.Error(), e.matcher.Description())
	}
	return nil
}

func (e *errorExpectation) Description() string {
	return fmt.Sprintf("error %s", e.matcher.Description())
}

type toolCallExpectation struct {
	toolName string
}

func (e *toolCallExpectation) Check(r *ScenarioResult) error {
	for _, tc := range r.ToolCalls {
		if tc.Name == e.toolName {
			return nil
		}
	}
	return fmt.Errorf("tool %q was not called", e.toolName)
}

func (e *toolCallExpectation) Description() string {
	return fmt.Sprintf("tool %q called", e.toolName)
}

type noToolCallsExpectation struct{}

func (e *noToolCallsExpectation) Check(r *ScenarioResult) error {
	if len(r.ToolCalls) > 0 {
		names := make([]string, len(r.ToolCalls))
		for i, tc := range r.ToolCalls {
			names[i] = tc.Name
		}
		return fmt.Errorf("expected no tool calls, got: %v", names)
	}
	return nil
}

func (e *noToolCallsExpectation) Description() string {
	return "no tool calls"
}

type eventExpectation struct {
	eventType core.EventType
}

func (e *eventExpectation) Check(r *ScenarioResult) error {
	for _, ev := range r.Events {
		if ev.Type == e.eventType {
			return nil
		}
	}
	return fmt.Errorf("event type %q was not emitted", e.eventType)
}

func (e *eventExpectation) Description() string {
	return fmt.Sprintf("event %q emitted", e.eventType)
}

type minDurationExpectation struct {
	min time.Duration
}

func (e *minDurationExpectation) Check(r *ScenarioResult) error {
	if r.Duration < e.min {
		return fmt.Errorf("duration %v is less than minimum %v", r.Duration, e.min)
	}
	return nil
}

func (e *minDurationExpectation) Description() string {
	return fmt.Sprintf("duration >= %v", e.min)
}

type maxDurationExpectation struct {
	max time.Duration
}

func (e *maxDurationExpectation) Check(r *ScenarioResult) error {
	if r.Duration > e.max {
		return fmt.Errorf("duration %v exceeds maximum %v", r.Duration, e.max)
	}
	return nil
}

func (e *maxDurationExpectation) Description() string {
	return fmt.Sprintf("duration <= %v", e.max)
}

// EventCollector collects events emitted during a scenario.
type EventCollector struct {
	mu     sync.RWMutex
	events []core.Event
}

// NewEventCollector creates a new event collector.
func NewEventCollector() *EventCollector {
	return &EventCollector{
		events: make([]core.Event, 0),
	}
}

// Collect adds an event to the collector.
func (c *EventCollector) Collect(event core.Event) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, event)
}

// Events returns all collected events.
func (c *EventCollector) Events() []core.Event {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([]core.Event, len(c.events))
	copy(result, c.events)
	return result
}

// EventTypes returns the types of all collected events.
func (c *EventCollector) EventTypes() []core.EventType {
	c.mu.RLock()
	defer c.mu.RUnlock()
	types := make([]core.EventType, len(c.events))
	for i, ev := range c.events {
		types[i] = ev.Type
	}
	return types
}

// HasEvent checks if an event of the given type was collected.
func (c *EventCollector) HasEvent(eventType core.EventType) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, ev := range c.events {
		if ev.Type == eventType {
			return true
		}
	}
	return false
}

// Count returns the number of collected events.
func (c *EventCollector) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.events)
}

// Reset clears all collected events.
func (c *EventCollector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = c.events[:0]
}
