// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

// Example 15: Testing Framework
//
// This example demonstrates how to use the Kairos testing framework
// to write tests for AI agents. It shows scenario definitions,
// mock providers, and assertion helpers.
package main

import (
	"context"
	"fmt"

	"github.com/jllopis/kairos/pkg/core"
	"github.com/jllopis/kairos/pkg/llm"
	ktesting "github.com/jllopis/kairos/pkg/testing"
)

func main() {
	fmt.Println("=== Kairos Testing Framework Demo ===")

	// Demo 1: ScenarioProvider for scripted responses
	fmt.Println("--- Demo 1: Scripted Provider ---")
	demoScriptedProvider()

	// Demo 2: Tool call builders
	fmt.Println("\n--- Demo 2: Tool Call Builders ---")
	demoToolCallBuilders()

	// Demo 3: Event collector
	fmt.Println("\n--- Demo 3: Event Collector ---")
	demoEventCollector()

	fmt.Println("\n=== Demo Complete ===")
	fmt.Println("\nSee examples/15-testing/example_test.go for full test examples.")
}

func demoScriptedProvider() {
	// Create a provider with scripted responses
	provider := ktesting.NewScenarioProvider().
		AddResponse("Hello! How can I help you?").
		AddToolCallResponse(
			ktesting.NewToolCall("search").
				WithArg("query", "weather").
				Build(),
		).
		AddResponse("The weather is sunny!")

	// Simulate conversation
	ctx := context.Background()

	resp1, _ := provider.Chat(ctx, llm.ChatRequest{})
	fmt.Printf("Turn 1: %s\n", resp1.Content)

	resp2, _ := provider.Chat(ctx, llm.ChatRequest{})
	fmt.Printf("Turn 2: Tool calls: %d (%s)\n", 
		len(resp2.ToolCalls), 
		resp2.ToolCalls[0].Function.Name)

	resp3, _ := provider.Chat(ctx, llm.ChatRequest{})
	fmt.Printf("Turn 3: %s\n", resp3.Content)

	fmt.Printf("Total calls: %d\n", provider.CallCount())
}

func demoToolCallBuilders() {
	// Build a tool call
	toolCall := ktesting.NewToolCall("get_weather").
		WithID("call_abc123").
		WithArg("city", "London").
		WithArg("unit", "celsius").
		Build()

	fmt.Printf("Tool: %s\n", toolCall.Function.Name)
	fmt.Printf("ID: %s\n", toolCall.ID)
	fmt.Printf("Args: %s\n", toolCall.Function.Arguments)

	// Build a tool definition
	toolDef := ktesting.NewToolDefinition("search").
		WithDescription("Search the web for information").
		WithParameter("query", "string", "Search query", true).
		WithParameter("limit", "integer", "Max results", false).
		Build()

	fmt.Printf("\nTool Definition: %s\n", toolDef.Function.Name)
	fmt.Printf("Description: %s\n", toolDef.Function.Description)
}

func demoEventCollector() {
	collector := ktesting.NewEventCollector()

	// In a real test, you'd connect this to an agent
	// agent.WithEventListener(collector.Collect)

	// Simulate some events
	collector.Collect(core.Event{Type: core.EventAgentThinking})
	collector.Collect(core.Event{Type: core.EventAgentTaskStarted})
	collector.Collect(core.Event{Type: core.EventAgentDelegation})
	collector.Collect(core.Event{Type: core.EventAgentTaskCompleted})

	fmt.Printf("Events collected: %d\n", collector.Count())
	fmt.Printf("Event types: %v\n", collector.EventTypes())
	fmt.Printf("Has 'task.started': %v\n", collector.HasEvent(core.EventAgentTaskStarted))
}
