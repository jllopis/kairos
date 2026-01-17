// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package testing

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jllopis/kairos/pkg/core"
	"github.com/jllopis/kairos/pkg/llm"
)

// mockAgent implements AgentRunner for testing.
type mockAgent struct {
	response string
	err      error
	delay    time.Duration
}

func (m *mockAgent) Run(ctx context.Context, input string) (string, error) {
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	return m.response, m.err
}

func TestScenarioBasic(t *testing.T) {
	agent := &mockAgent{response: "Hello, World!"}

	scenario := NewScenario("basic test").
		WithInput("Hi").
		ExpectNoError().
		ExpectOutput(Contains("Hello"))

	result := scenario.Run(t, agent)
	result.Assert(t, scenario)
}

func TestScenarioWithError(t *testing.T) {
	agent := &mockAgent{err: errors.New("something went wrong")}

	scenario := NewScenario("error test").
		WithInput("Hi").
		ExpectError(Contains("went wrong"))

	result := scenario.Run(t, agent)
	result.Assert(t, scenario)
}

func TestScenarioDuration(t *testing.T) {
	agent := &mockAgent{response: "ok", delay: 50 * time.Millisecond}

	scenario := NewScenario("duration test").
		WithInput("Hi").
		WithTimeout(1 * time.Second).
		ExpectNoError().
		ExpectMinDuration(40 * time.Millisecond).
		ExpectMaxDuration(200 * time.Millisecond)

	result := scenario.Run(t, agent)
	result.Assert(t, scenario)
}

func TestScenarioTimeout(t *testing.T) {
	agent := &mockAgent{response: "ok", delay: 500 * time.Millisecond}

	scenario := NewScenario("timeout test").
		WithInput("Hi").
		WithTimeout(50 * time.Millisecond).
		ExpectError(Contains("context deadline"))

	result := scenario.Run(t, agent)
	result.Assert(t, scenario)
}

func TestStringMatchers(t *testing.T) {
	tests := []struct {
		name    string
		matcher StringMatcher
		input   string
		match   bool
	}{
		{"contains match", Contains("world"), "hello world", true},
		{"contains no match", Contains("foo"), "hello world", false},
		{"equals match", Equals("hello"), "hello", true},
		{"equals no match", Equals("hello"), "Hello", false},
		{"prefix match", HasPrefix("hello"), "hello world", true},
		{"prefix no match", HasPrefix("world"), "hello world", false},
		{"suffix match", HasSuffix("world"), "hello world", true},
		{"suffix no match", HasSuffix("hello"), "hello world", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.matcher.Match(tc.input); got != tc.match {
				t.Errorf("expected match=%v, got %v", tc.match, got)
			}
		})
	}
}

func TestScenarioProvider(t *testing.T) {
	provider := NewScenarioProvider().
		AddResponse("First response").
		AddResponse("Second response")

	// First call
	resp1, err := provider.Chat(context.Background(), llm.ChatRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp1.Content != "First response" {
		t.Errorf("expected 'First response', got %q", resp1.Content)
	}

	// Second call
	resp2, err := provider.Chat(context.Background(), llm.ChatRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp2.Content != "Second response" {
		t.Errorf("expected 'Second response', got %q", resp2.Content)
	}

	// Third call should error
	_, err = provider.Chat(context.Background(), llm.ChatRequest{})
	if err == nil {
		t.Error("expected error for third call")
	}

	// Verify call count
	if provider.CallCount() != 3 {
		t.Errorf("expected 3 calls, got %d", provider.CallCount())
	}
}

func TestScenarioProviderToolCalls(t *testing.T) {
	toolCall := NewToolCall("search").
		WithID("call_123").
		WithArg("query", "test").
		Build()

	provider := NewScenarioProvider().
		AddToolCallResponse(toolCall).
		AddResponse("Search complete")

	// First call returns tool call
	resp1, err := provider.Chat(context.Background(), llm.ChatRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp1.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(resp1.ToolCalls))
	}
	if resp1.ToolCalls[0].Function.Name != "search" {
		t.Errorf("expected tool 'search', got %q", resp1.ToolCalls[0].Function.Name)
	}

	// Second call returns content
	resp2, err := provider.Chat(context.Background(), llm.ChatRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp2.Content != "Search complete" {
		t.Errorf("expected 'Search complete', got %q", resp2.Content)
	}
}

func TestScenarioProviderRequestCapture(t *testing.T) {
	provider := NewScenarioProvider().
		AddResponse("ok")

	req := llm.ChatRequest{
		Model: "test-model",
		Messages: []llm.Message{
			{Role: llm.RoleUser, Content: "Hello"},
		},
	}

	_, _ = provider.Chat(context.Background(), req)

	captured := provider.LastRequest()
	if captured == nil {
		t.Fatal("expected captured request")
	}
	if captured.Model != "test-model" {
		t.Errorf("expected model 'test-model', got %q", captured.Model)
	}
	if len(captured.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(captured.Messages))
	}
}

func TestToolCallBuilder(t *testing.T) {
	tc := NewToolCall("get_weather").
		WithID("call_abc").
		WithArg("city", "London").
		WithArg("unit", "celsius").
		Build()

	if tc.Function.Name != "get_weather" {
		t.Errorf("expected name 'get_weather', got %q", tc.Function.Name)
	}
	if tc.ID != "call_abc" {
		t.Errorf("expected ID 'call_abc', got %q", tc.ID)
	}
	if tc.Type != llm.ToolTypeFunction {
		t.Errorf("expected type 'function', got %q", tc.Type)
	}
	// Args should be JSON
	if tc.Function.Arguments == "" {
		t.Error("expected arguments to be set")
	}
}

func TestToolDefinitionBuilder(t *testing.T) {
	tool := NewToolDefinition("search").
		WithDescription("Search the web").
		WithParameter("query", "string", "Search query", true).
		WithParameter("limit", "integer", "Max results", false).
		Build()

	if tool.Function.Name != "search" {
		t.Errorf("expected name 'search', got %q", tool.Function.Name)
	}
	if tool.Function.Description != "Search the web" {
		t.Errorf("expected description, got %q", tool.Function.Description)
	}
	if tool.Type != llm.ToolTypeFunction {
		t.Errorf("expected type 'function', got %q", tool.Type)
	}
}

func TestEventCollector(t *testing.T) {
	collector := NewEventCollector()

	collector.Collect(core.Event{Type: core.EventAgentThinking})
	collector.Collect(core.Event{Type: core.EventAgentTaskStarted})
	collector.Collect(core.Event{Type: core.EventAgentTaskCompleted})

	if collector.Count() != 3 {
		t.Errorf("expected 3 events, got %d", collector.Count())
	}

	if !collector.HasEvent(core.EventAgentTaskStarted) {
		t.Error("expected to find 'task.started' event")
	}

	if collector.HasEvent(core.EventType("nonexistent")) {
		t.Error("should not find 'nonexistent' event")
	}

	types := collector.EventTypes()
	if len(types) != 3 {
		t.Errorf("expected 3 types, got %d", len(types))
	}

	collector.Reset()
	if collector.Count() != 0 {
		t.Errorf("expected 0 events after reset, got %d", collector.Count())
	}
}

func TestAssertions(t *testing.T) {
	// Use a sub-test to capture failures
	t.Run("passing assertions", func(t *testing.T) {
		a := NewAssertions(t)
		
		a.AssertEqual(1, 1, "equal")
		a.AssertNotEqual(1, 2, "not equal")
		a.AssertTrue(true, "true")
		a.AssertFalse(false, "false")
		a.AssertContains("hello world", "world", "contains")
		a.AssertNotContains("hello", "world", "not contains")
		a.AssertNoError(nil, "no error")
		a.AssertError(errors.New("oops"), "error")
		
		if a.Failed() {
			t.Error("assertions should not have failed")
		}
	})
}

func TestRequestAssertions(t *testing.T) {
	a := NewAssertions(t)
	
	req := &llm.ChatRequest{
		Model: "gpt-4",
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: "You are a helpful assistant"},
			{Role: llm.RoleUser, Content: "Hello"},
		},
		Tools: []llm.Tool{
			NewToolDefinition("search").Build(),
		},
	}
	
	a.AssertRequest(req).
		HasModel("gpt-4").
		HasMessageCount(2).
		HasToolCount(1).
		HasSystemMessage("helpful").
		HasUserMessage("Hello").
		HasTool("search")
}

func TestResponseAssertions(t *testing.T) {
	a := NewAssertions(t)
	
	resp := &llm.ChatResponse{
		Content: "Hello, how can I help?",
		ToolCalls: []llm.ToolCall{
			NewToolCall("search").WithArg("q", "test").Build(),
		},
	}
	
	a.AssertResponse(resp).
		HasContent("help").
		HasToolCalls().
		HasToolCallCount(1).
		HasToolCallNamed("search")
}
