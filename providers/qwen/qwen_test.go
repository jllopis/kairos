// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package qwen

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jllopis/kairos/pkg/llm"
)

func TestProviderImplementsInterface(t *testing.T) {
	var _ llm.Provider = (*Provider)(nil)
}

func TestNewProvider(t *testing.T) {
	p := New("test-api-key")
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
	if p.model != "qwen-turbo" {
		t.Errorf("expected model qwen-turbo, got %s", p.model)
	}
	if p.baseURL != DefaultBaseURL {
		t.Errorf("expected baseURL %s, got %s", DefaultBaseURL, p.baseURL)
	}
}

func TestWithModel(t *testing.T) {
	p := New("test-key", WithModel("qwen-max"))
	if p.model != "qwen-max" {
		t.Errorf("expected model qwen-max, got %s", p.model)
	}
}

func TestWithBaseURL(t *testing.T) {
	customURL := "https://custom.api.com/v1"
	p := New("test-key", WithBaseURL(customURL))
	if p.baseURL != customURL {
		t.Errorf("expected baseURL %s, got %s", customURL, p.baseURL)
	}
}

func TestConvertMessages(t *testing.T) {
	messages := []llm.Message{
		{Role: llm.RoleSystem, Content: "You are helpful"},
		{Role: llm.RoleUser, Content: "Hello"},
		{Role: llm.RoleAssistant, Content: "Hi there"},
	}

	result := convertMessages(messages)
	if len(result) != 3 {
		t.Errorf("expected 3 messages, got %d", len(result))
	}
	if result[0].Role != "system" {
		t.Errorf("expected role system, got %s", result[0].Role)
	}
	if result[1].Role != "user" {
		t.Errorf("expected role user, got %s", result[1].Role)
	}
	if result[2].Role != "assistant" {
		t.Errorf("expected role assistant, got %s", result[2].Role)
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
	if result[0].Type != "function" {
		t.Errorf("expected type function, got %s", result[0].Type)
	}
	if result[0].Function.Name != "get_weather" {
		t.Errorf("expected name get_weather, got %s", result[0].Function.Name)
	}
}

func TestConvertResponse(t *testing.T) {
	resp := &openAIResponse{
		ID: "chatcmpl-123",
		Choices: []struct {
			Index   int `json:"index"`
			Message struct {
				Role      string           `json:"role"`
				Content   string           `json:"content"`
				ToolCalls []openAIToolCall `json:"tool_calls,omitempty"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		}{
			{
				Index: 0,
				Message: struct {
					Role      string           `json:"role"`
					Content   string           `json:"content"`
					ToolCalls []openAIToolCall `json:"tool_calls,omitempty"`
				}{
					Role:    "assistant",
					Content: "Hello there!",
				},
				FinishReason: "stop",
			},
		},
		Usage: struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		}{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}

	result := convertResponse(resp)
	if result.Content != "Hello there!" {
		t.Errorf("expected content 'Hello there!', got %s", result.Content)
	}
	if result.Usage.TotalTokens != 15 {
		t.Errorf("expected total tokens 15, got %d", result.Usage.TotalTokens)
	}
}

func TestStreamingProviderInterface(t *testing.T) {
	var _ llm.StreamingProvider = (*Provider)(nil)
}

func TestChatStream(t *testing.T) {
	// Create a mock server that returns SSE streaming response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("Expected /chat/completions, got %s", r.URL.Path)
		}

		// Send SSE streaming response
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		// Stream chunks
		events := []string{
			`data: {"id":"chatcmpl-1","choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"}}]}`,
			`data: {"id":"chatcmpl-1","choices":[{"index":0,"delta":{"content":" world"}}]}`,
			`data: {"id":"chatcmpl-1","choices":[{"index":0,"delta":{"content":"!"}}],"usage":{"prompt_tokens":10,"completion_tokens":3,"total_tokens":13}}`,
			`data: [DONE]`,
		}

		for _, event := range events {
			w.Write([]byte(event + "\n\n"))
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}))
	defer server.Close()

	// Create provider with mock server
	provider := New("test-key", WithBaseURL(server.URL))

	// Test streaming
	ctx := context.Background()
	stream, err := provider.ChatStream(ctx, llm.ChatRequest{
		Model:    "qwen-turbo",
		Messages: []llm.Message{{Role: llm.RoleUser, Content: "Hi"}},
	})
	if err != nil {
		t.Fatalf("ChatStream failed: %v", err)
	}

	// Collect chunks
	var content string
	var gotDone bool
	var usage *llm.Usage

	for chunk := range stream {
		if chunk.Error != nil {
			t.Fatalf("Stream error: %v", chunk.Error)
		}
		content += chunk.Content
		if chunk.Done {
			gotDone = true
			usage = chunk.Usage
		}
	}

	// Verify results
	if content != "Hello world!" {
		t.Errorf("Expected 'Hello world!', got '%s'", content)
	}
	if !gotDone {
		t.Error("Expected done=true in final chunk")
	}
	if usage == nil {
		t.Error("Expected usage in final chunk")
	} else if usage.TotalTokens != 13 {
		t.Errorf("Expected 13 total tokens, got %d", usage.TotalTokens)
	}
}
