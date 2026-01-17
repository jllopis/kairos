// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package openai

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
	if p.model != "gpt-5-mini" {
		t.Errorf("expected model gpt-5-mini, got %s", p.model)
	}
}

func TestWithModel(t *testing.T) {
	p := New(WithModel("gpt-4-turbo"))
	if p.model != "gpt-4-turbo" {
		t.Errorf("expected model gpt-4-turbo, got %s", p.model)
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
			name: "system message",
			msg:  llm.Message{Role: llm.RoleSystem, Content: "You are helpful"},
		},
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
			msg:  llm.Message{Role: llm.RoleTool, Content: "result", ToolCallID: "call_123"},
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
