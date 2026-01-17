// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"testing"
	"time"

	"github.com/jllopis/kairos/pkg/llm"
	ktesting "github.com/jllopis/kairos/pkg/testing"
)

// ExampleAgent is a simple agent for testing demonstration.
type ExampleAgent struct {
	provider llm.Provider
}

func (a *ExampleAgent) Run(ctx context.Context, input string) (string, error) {
	resp, err := a.provider.Chat(ctx, llm.ChatRequest{
		Model: "test-model",
		Messages: []llm.Message{
			{Role: llm.RoleUser, Content: input},
		},
	})
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

func TestAgentBasicResponse(t *testing.T) {
	// Setup: Create a scripted provider
	provider := ktesting.NewScenarioProvider().
		AddResponse("Hello! I'm your assistant.")

	agent := &ExampleAgent{provider: provider}

	// Define scenario
	scenario := ktesting.NewScenario("basic greeting").
		WithInput("Hello").
		WithTimeout(5 * time.Second).
		ExpectNoError().
		ExpectOutput(ktesting.Contains("assistant"))

	// Run and assert
	result := scenario.Run(t, agent)
	result.Assert(t, scenario)

	// Additional assertions
	a := ktesting.NewAssertions(t)
	a.AssertEqual(1, provider.CallCount(), "call count")
}

func TestAgentWithToolCalls(t *testing.T) {
	// Setup: Provider returns a tool call, then a response
	searchTool := ktesting.NewToolCall("search").
		WithID("call_1").
		WithArg("query", "weather in London").
		Build()

	provider := ktesting.NewScenarioProvider().
		AddToolCallResponse(searchTool).
		AddResponse("The weather in London is cloudy.")

	// First call returns tool call
	resp1, err := provider.Chat(context.Background(), llm.ChatRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	a := ktesting.NewAssertions(t)
	a.AssertResponse(resp1).
		HasToolCalls().
		HasToolCallCount(1).
		HasToolCallNamed("search")

	// Second call returns content
	resp2, err := provider.Chat(context.Background(), llm.ChatRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	a.AssertResponse(resp2).
		HasNoToolCalls().
		HasContent("cloudy")
}

func TestRequestCapture(t *testing.T) {
	provider := ktesting.NewScenarioProvider().
		AddResponse("ok")

	agent := &ExampleAgent{provider: provider}

	_, _ = agent.Run(context.Background(), "What's 2+2?")

	// Verify the request was captured correctly
	a := ktesting.NewAssertions(t)
	a.AssertRequest(provider.LastRequest()).
		HasModel("test-model").
		HasMessageCount(1).
		HasUserMessage("2+2")
}

func TestToolDefinitionBuilder(t *testing.T) {
	// Build a tool definition
	tool := ktesting.NewToolDefinition("calculator").
		WithDescription("Perform calculations").
		WithParameter("expression", "string", "Math expression", true).
		WithParameter("precision", "integer", "Decimal places", false).
		Build()

	a := ktesting.NewAssertions(t)
	a.AssertEqual("calculator", tool.Function.Name, "tool name")
	a.AssertContains(tool.Function.Description, "calculation", "description")
	a.AssertEqual(llm.ToolTypeFunction, tool.Type, "tool type")
}

func TestScenarioWithSetupTeardown(t *testing.T) {
	setupCalled := false
	teardownCalled := false

	provider := ktesting.NewScenarioProvider().
		AddResponse("ok")

	agent := &ExampleAgent{provider: provider}

	scenario := ktesting.NewScenario("with lifecycle").
		WithInput("test").
		WithSetup(func() error {
			setupCalled = true
			return nil
		}).
		WithTeardown(func() error {
			teardownCalled = true
			return nil
		}).
		ExpectNoError()

	_ = scenario.Run(t, agent)

	if !setupCalled {
		t.Error("setup was not called")
	}
	if !teardownCalled {
		t.Error("teardown was not called")
	}
}

func TestStringMatchers(t *testing.T) {
	tests := []struct {
		name    string
		matcher ktesting.StringMatcher
		input   string
		want    bool
	}{
		{"contains yes", ktesting.Contains("world"), "hello world", true},
		{"contains no", ktesting.Contains("foo"), "hello world", false},
		{"equals yes", ktesting.Equals("hello"), "hello", true},
		{"equals no", ktesting.Equals("Hello"), "hello", false},
		{"prefix yes", ktesting.HasPrefix("hello"), "hello world", true},
		{"prefix no", ktesting.HasPrefix("world"), "hello world", false},
		{"suffix yes", ktesting.HasSuffix("world"), "hello world", true},
		{"suffix no", ktesting.HasSuffix("hello"), "hello world", false},
		{"regex yes", ktesting.Regex(`\d+`), "test123", true},
		{"regex no", ktesting.Regex(`^\d+$`), "test123", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.matcher.Match(tc.input)
			if got != tc.want {
				t.Errorf("Match(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}
