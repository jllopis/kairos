// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package anthropic

import (
	"testing"

	"github.com/jllopis/kairos/pkg/llm"
)

func TestProviderImplementsInterface(t *testing.T) {
	var _ llm.Provider = (*Provider)(nil)
}

func TestNewProvider(t *testing.T) {
	p := New()
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
	if p.model != "claude-sonnet-4-20250514" {
		t.Errorf("expected model claude-sonnet-4-20250514, got %s", p.model)
	}
	if p.maxTokens != 4096 {
		t.Errorf("expected maxTokens 4096, got %d", p.maxTokens)
	}
}

func TestWithModel(t *testing.T) {
	p := New(WithModel("claude-opus-4-20250514"))
	if p.model != "claude-opus-4-20250514" {
		t.Errorf("expected model claude-opus-4-20250514, got %s", p.model)
	}
}

func TestWithMaxTokens(t *testing.T) {
	p := New(WithMaxTokens(8192))
	if p.maxTokens != 8192 {
		t.Errorf("expected maxTokens 8192, got %d", p.maxTokens)
	}
}

func TestNewWithAPIKey(t *testing.T) {
	p := NewWithAPIKey("test-key")
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
}

func TestConvertMessages(t *testing.T) {
	tests := []struct {
		name string
		msg  llm.Message
	}{
		{
			name: "user message",
			msg:  llm.Message{Role: llm.RoleUser, Content: "Hello"},
		},
		{
			name: "assistant message",
			msg:  llm.Message{Role: llm.RoleAssistant, Content: "Hi there"},
		},
		{
			name: "tool message",
			msg:  llm.Message{Role: llm.RoleTool, Content: "result", ToolCallID: "toolu_123"},
		},
		{
			name: "assistant with tool calls",
			msg: llm.Message{
				Role: llm.RoleAssistant,
				ToolCalls: []llm.ToolCall{
					{
						ID:   "toolu_123",
						Type: llm.ToolTypeFunction,
						Function: llm.FunctionCall{
							Name:      "get_weather",
							Arguments: `{"location":"Paris"}`,
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify conversion doesn't panic
			_ = convertMessage(tt.msg)
		})
	}
}

func TestConvertTool(t *testing.T) {
	tool := llm.Tool{
		Type: llm.ToolTypeFunction,
		Function: llm.FunctionDef{
			Name:        "get_weather",
			Description: "Get weather for a location",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"location": map[string]interface{}{
						"type":        "string",
						"description": "The city name",
					},
				},
				"required": []string{"location"},
			},
		},
	}

	// Just verify conversion doesn't panic
	_ = convertTool(tool)
}
