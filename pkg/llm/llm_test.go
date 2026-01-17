package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMockProvider(t *testing.T) {
	mock := &MockProvider{Response: "Hello world"}
	resp, err := mock.Chat(context.Background(), ChatRequest{
		Messages: []Message{{Role: RoleUser, Content: "Hi"}},
	})
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	if resp.Content != "Hello world" {
		t.Errorf("Expected 'Hello world', got '%s'", resp.Content)
	}
}

func TestOllamaProviderChatStream(t *testing.T) {
	// Create a mock server that returns NDJSON streaming response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Errorf("Expected /api/chat, got %s", r.URL.Path)
		}

		// Verify request
		var req ollamaRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}
		if !req.Stream {
			t.Error("Expected stream=true in request")
		}

		// Send NDJSON streaming response
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.WriteHeader(http.StatusOK)

		// Stream chunks
		chunks := []ollamaStreamEvent{
			{Model: "llama3", Message: Message{Role: RoleAssistant, Content: "Hello"}, Done: false},
			{Model: "llama3", Message: Message{Role: RoleAssistant, Content: " world"}, Done: false},
			{Model: "llama3", Message: Message{Role: RoleAssistant, Content: "!"}, Done: false},
			{Model: "llama3", Done: true, PromptEvalCount: 10, EvalCount: 5},
		}

		for _, chunk := range chunks {
			data, _ := json.Marshal(chunk)
			w.Write(data)
			w.Write([]byte("\n"))
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}))
	defer server.Close()

	// Create provider with mock server
	provider := NewOllama(server.URL)

	// Test streaming
	ctx := context.Background()
	stream, err := provider.ChatStream(ctx, ChatRequest{
		Model:    "llama3",
		Messages: []Message{{Role: RoleUser, Content: "Hi"}},
	})
	if err != nil {
		t.Fatalf("ChatStream failed: %v", err)
	}

	// Collect chunks
	var content string
	var gotDone bool
	var usage *Usage

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
	} else if usage.TotalTokens != 15 {
		t.Errorf("Expected 15 total tokens, got %d", usage.TotalTokens)
	}
}
