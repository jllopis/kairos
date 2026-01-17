// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package gemini

import (
	"testing"

	"github.com/jllopis/kairos/pkg/llm"
)

func TestProviderImplementsInterface(t *testing.T) {
	var _ llm.Provider = (*Provider)(nil)
}

func TestWithModel(t *testing.T) {
	// Test option function
	opt := WithModel("gemini-1.5-pro")
	p := &Provider{model: "gemini-2.0-flash"}
	opt(p)
	if p.model != "gemini-1.5-pro" {
		t.Errorf("expected model gemini-1.5-pro, got %s", p.model)
	}
}

func TestConvertMessages(t *testing.T) {
	messages := []llm.Message{
		{Role: llm.RoleSystem, Content: "You are helpful"},
		{Role: llm.RoleUser, Content: "Hello"},
		{Role: llm.RoleAssistant, Content: "Hi there"},
	}

	contents, systemInstruction := convertMessages(messages)

	if systemInstruction != "You are helpful" {
		t.Errorf("expected system instruction 'You are helpful', got %s", systemInstruction)
	}

	// Should have 2 contents (user and assistant), system is extracted
	if len(contents) != 2 {
		t.Errorf("expected 2 contents, got %d", len(contents))
	}
}

func TestConvertTools(t *testing.T) {
	tools := []llm.Tool{
		{
			Type: llm.ToolTypeFunction,
			Function: llm.FunctionDef{
				Name:        "get_weather",
				Description: "Get weather for a location",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"location": map[string]interface{}{"type": "string"},
					},
				},
			},
		},
	}

	result := convertTools(tools)
	if len(result) != 1 {
		t.Errorf("expected 1 tool, got %d", len(result))
	}
	if result[0].Name != "get_weather" {
		t.Errorf("expected name get_weather, got %s", result[0].Name)
	}
}

func TestClose(t *testing.T) {
	p := &Provider{}
	err := p.Close()
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}
