// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package testing

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/jllopis/kairos/pkg/llm"
)

// Assertions provides assertion helpers for testing.
type Assertions struct {
	t      *testing.T
	failed bool
}

// NewAssertions creates a new assertions helper.
func NewAssertions(t *testing.T) *Assertions {
	return &Assertions{t: t}
}

// Failed returns true if any assertion has failed.
func (a *Assertions) Failed() bool {
	return a.failed
}

// AssertEqual asserts that two values are equal.
func (a *Assertions) AssertEqual(expected, actual any, msg string) {
	a.t.Helper()
	if expected != actual {
		a.t.Errorf("%s: expected %v, got %v", msg, expected, actual)
		a.failed = true
	}
}

// AssertNotEqual asserts that two values are not equal.
func (a *Assertions) AssertNotEqual(expected, actual any, msg string) {
	a.t.Helper()
	if expected == actual {
		a.t.Errorf("%s: expected not %v, got %v", msg, expected, actual)
		a.failed = true
	}
}

// AssertNil asserts that the value is nil.
func (a *Assertions) AssertNil(value any, msg string) {
	a.t.Helper()
	if value != nil {
		a.t.Errorf("%s: expected nil, got %v", msg, value)
		a.failed = true
	}
}

// AssertNotNil asserts that the value is not nil.
func (a *Assertions) AssertNotNil(value any, msg string) {
	a.t.Helper()
	if value == nil {
		a.t.Errorf("%s: expected non-nil value", msg)
		a.failed = true
	}
}

// AssertTrue asserts that the value is true.
func (a *Assertions) AssertTrue(value bool, msg string) {
	a.t.Helper()
	if !value {
		a.t.Errorf("%s: expected true", msg)
		a.failed = true
	}
}

// AssertFalse asserts that the value is false.
func (a *Assertions) AssertFalse(value bool, msg string) {
	a.t.Helper()
	if value {
		a.t.Errorf("%s: expected false", msg)
		a.failed = true
	}
}

// AssertContains asserts that the string contains the substring.
func (a *Assertions) AssertContains(s, substr, msg string) {
	a.t.Helper()
	if !strings.Contains(s, substr) {
		a.t.Errorf("%s: %q does not contain %q", msg, s, substr)
		a.failed = true
	}
}

// AssertNotContains asserts that the string does not contain the substring.
func (a *Assertions) AssertNotContains(s, substr, msg string) {
	a.t.Helper()
	if strings.Contains(s, substr) {
		a.t.Errorf("%s: %q should not contain %q", msg, s, substr)
		a.failed = true
	}
}

// AssertError asserts that the error is not nil.
func (a *Assertions) AssertError(err error, msg string) {
	a.t.Helper()
	if err == nil {
		a.t.Errorf("%s: expected error, got nil", msg)
		a.failed = true
	}
}

// AssertNoError asserts that the error is nil.
func (a *Assertions) AssertNoError(err error, msg string) {
	a.t.Helper()
	if err != nil {
		a.t.Errorf("%s: unexpected error: %v", msg, err)
		a.failed = true
	}
}

// AssertErrorContains asserts that the error message contains the substring.
func (a *Assertions) AssertErrorContains(err error, substr, msg string) {
	a.t.Helper()
	if err == nil {
		a.t.Errorf("%s: expected error containing %q, got nil", msg, substr)
		a.failed = true
		return
	}
	if !strings.Contains(err.Error(), substr) {
		a.t.Errorf("%s: error %q does not contain %q", msg, err.Error(), substr)
		a.failed = true
	}
}

// AssertLen asserts the length of a slice or map.
func (a *Assertions) AssertLen(value any, expected int, msg string) {
	a.t.Helper()
	var length int
	switch v := value.(type) {
	case string:
		length = len(v)
	case []any:
		length = len(v)
	case []string:
		length = len(v)
	case []llm.ToolCall:
		length = len(v)
	case []llm.Message:
		length = len(v)
	case map[string]any:
		length = len(v)
	default:
		a.t.Errorf("%s: cannot get length of %T", msg, value)
		a.failed = true
		return
	}
	if length != expected {
		a.t.Errorf("%s: expected length %d, got %d", msg, expected, length)
		a.failed = true
	}
}

// RequestAssertions provides assertion helpers for LLM requests.
type RequestAssertions struct {
	*Assertions
	req *llm.ChatRequest
}

// AssertRequest creates request assertions for the given request.
func (a *Assertions) AssertRequest(req *llm.ChatRequest) *RequestAssertions {
	a.t.Helper()
	if req == nil {
		a.t.Error("request is nil")
		a.failed = true
		return &RequestAssertions{Assertions: a, req: &llm.ChatRequest{}}
	}
	return &RequestAssertions{Assertions: a, req: req}
}

// HasModel asserts the request uses the given model.
func (r *RequestAssertions) HasModel(model string) *RequestAssertions {
	r.t.Helper()
	if r.req.Model != model {
		r.t.Errorf("expected model %q, got %q", model, r.req.Model)
		r.failed = true
	}
	return r
}

// HasMessageCount asserts the number of messages in the request.
func (r *RequestAssertions) HasMessageCount(count int) *RequestAssertions {
	r.t.Helper()
	if len(r.req.Messages) != count {
		r.t.Errorf("expected %d messages, got %d", count, len(r.req.Messages))
		r.failed = true
	}
	return r
}

// HasToolCount asserts the number of tools in the request.
func (r *RequestAssertions) HasToolCount(count int) *RequestAssertions {
	r.t.Helper()
	if len(r.req.Tools) != count {
		r.t.Errorf("expected %d tools, got %d", count, len(r.req.Tools))
		r.failed = true
	}
	return r
}

// HasSystemMessage asserts a system message exists with the given content.
func (r *RequestAssertions) HasSystemMessage(contains string) *RequestAssertions {
	r.t.Helper()
	for _, msg := range r.req.Messages {
		if msg.Role == llm.RoleSystem && strings.Contains(msg.Content, contains) {
			return r
		}
	}
	r.t.Errorf("no system message containing %q found", contains)
	r.failed = true
	return r
}

// HasUserMessage asserts a user message exists with the given content.
func (r *RequestAssertions) HasUserMessage(contains string) *RequestAssertions {
	r.t.Helper()
	for _, msg := range r.req.Messages {
		if msg.Role == llm.RoleUser && strings.Contains(msg.Content, contains) {
			return r
		}
	}
	r.t.Errorf("no user message containing %q found", contains)
	r.failed = true
	return r
}

// HasTool asserts a tool with the given name exists.
func (r *RequestAssertions) HasTool(name string) *RequestAssertions {
	r.t.Helper()
	for _, tool := range r.req.Tools {
		if tool.Function.Name == name {
			return r
		}
	}
	r.t.Errorf("tool %q not found in request", name)
	r.failed = true
	return r
}

// ResponseAssertions provides assertion helpers for LLM responses.
type ResponseAssertions struct {
	*Assertions
	resp *llm.ChatResponse
}

// AssertResponse creates response assertions for the given response.
func (a *Assertions) AssertResponse(resp *llm.ChatResponse) *ResponseAssertions {
	a.t.Helper()
	if resp == nil {
		a.t.Error("response is nil")
		a.failed = true
		return &ResponseAssertions{Assertions: a, resp: &llm.ChatResponse{}}
	}
	return &ResponseAssertions{Assertions: a, resp: resp}
}

// HasContent asserts the response has content containing the substring.
func (r *ResponseAssertions) HasContent(contains string) *ResponseAssertions {
	r.t.Helper()
	if !strings.Contains(r.resp.Content, contains) {
		r.t.Errorf("response content %q does not contain %q", r.resp.Content, contains)
		r.failed = true
	}
	return r
}

// HasNoContent asserts the response has no content.
func (r *ResponseAssertions) HasNoContent() *ResponseAssertions {
	r.t.Helper()
	if r.resp.Content != "" {
		r.t.Errorf("expected no content, got %q", r.resp.Content)
		r.failed = true
	}
	return r
}

// HasToolCalls asserts the response has tool calls.
func (r *ResponseAssertions) HasToolCalls() *ResponseAssertions {
	r.t.Helper()
	if len(r.resp.ToolCalls) == 0 {
		r.t.Error("expected tool calls, got none")
		r.failed = true
	}
	return r
}

// HasNoToolCalls asserts the response has no tool calls.
func (r *ResponseAssertions) HasNoToolCalls() *ResponseAssertions {
	r.t.Helper()
	if len(r.resp.ToolCalls) > 0 {
		r.t.Errorf("expected no tool calls, got %d", len(r.resp.ToolCalls))
		r.failed = true
	}
	return r
}

// HasToolCallCount asserts the number of tool calls.
func (r *ResponseAssertions) HasToolCallCount(count int) *ResponseAssertions {
	r.t.Helper()
	if len(r.resp.ToolCalls) != count {
		r.t.Errorf("expected %d tool calls, got %d", count, len(r.resp.ToolCalls))
		r.failed = true
	}
	return r
}

// HasToolCallNamed asserts a tool call with the given name exists.
func (r *ResponseAssertions) HasToolCallNamed(name string) *ResponseAssertions {
	r.t.Helper()
	for _, tc := range r.resp.ToolCalls {
		if tc.Function.Name == name {
			return r
		}
	}
	r.t.Errorf("tool call %q not found", name)
	r.failed = true
	return r
}

// ScenarioResultAssertions provides assertions for scenario results.
type ScenarioResultAssertions struct {
	*Assertions
	result *ScenarioResult
}

// AssertScenarioResult creates assertions for a scenario result.
func (a *Assertions) AssertScenarioResult(result *ScenarioResult) *ScenarioResultAssertions {
	a.t.Helper()
	if result == nil {
		a.t.Error("scenario result is nil")
		a.failed = true
		return &ScenarioResultAssertions{Assertions: a, result: &ScenarioResult{}}
	}
	return &ScenarioResultAssertions{Assertions: a, result: result}
}

// Succeeded asserts the scenario completed without error.
func (s *ScenarioResultAssertions) Succeeded() *ScenarioResultAssertions {
	s.t.Helper()
	if s.result.Error != nil {
		s.t.Errorf("expected success, got error: %v", s.result.Error)
		s.failed = true
	}
	return s
}

// Failed asserts the scenario failed with an error.
func (s *ScenarioResultAssertions) Failed() *ScenarioResultAssertions {
	s.t.Helper()
	if s.result.Error == nil {
		s.t.Error("expected failure, got success")
		s.failed = true
	}
	return s
}

// OutputContains asserts the output contains the substring.
func (s *ScenarioResultAssertions) OutputContains(substr string) *ScenarioResultAssertions {
	s.t.Helper()
	if !strings.Contains(s.result.Output, substr) {
		s.t.Errorf("output %q does not contain %q", s.result.Output, substr)
		s.failed = true
	}
	return s
}

// OutputEquals asserts the output equals the expected string.
func (s *ScenarioResultAssertions) OutputEquals(expected string) *ScenarioResultAssertions {
	s.t.Helper()
	if s.result.Output != expected {
		s.t.Errorf("expected output %q, got %q", expected, s.result.Output)
		s.failed = true
	}
	return s
}

// Quick assertion functions for common patterns

// RequireNoError fails the test immediately if err is not nil.
func RequireNoError(t *testing.T, err error, msg string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", msg, err)
	}
}

// RequireEqual fails the test immediately if values are not equal.
func RequireEqual(t *testing.T, expected, actual any, msg string) {
	t.Helper()
	if expected != actual {
		t.Fatalf("%s: expected %v, got %v", msg, expected, actual)
	}
}

// RequireNotNil fails the test immediately if value is nil.
func RequireNotNil(t *testing.T, value any, msg string) {
	t.Helper()
	if value == nil {
		t.Fatalf("%s: expected non-nil value", msg)
	}
}

// AssertToolCallArgs extracts and validates tool call arguments.
func AssertToolCallArgs(t *testing.T, tc llm.ToolCall, expectedName string) map[string]any {
	t.Helper()
	if tc.Function.Name != expectedName {
		t.Errorf("expected tool %q, got %q", expectedName, tc.Function.Name)
	}
	
	var args map[string]any
	if tc.Function.Arguments != "" {
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			t.Errorf("failed to parse tool arguments: %v", err)
			return nil
		}
	}
	return args
}

// FormatToolCalls formats tool calls for error messages.
func FormatToolCalls(calls []llm.ToolCall) string {
	if len(calls) == 0 {
		return "(none)"
	}
	names := make([]string, len(calls))
	for i, tc := range calls {
		names[i] = tc.Function.Name
	}
	return fmt.Sprintf("[%s]", strings.Join(names, ", "))
}
