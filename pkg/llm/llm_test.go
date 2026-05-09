package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
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

func TestNewOllama_DefaultTimeout(t *testing.T) {
	t.Setenv(envOllamaTimeout, "")
	p := NewOllama("")
	if p.client.Timeout != defaultOllamaTimeout {
		t.Errorf("default timeout = %v, want %v", p.client.Timeout, defaultOllamaTimeout)
	}
	if p.baseURL != "http://localhost:11434" {
		t.Errorf("default baseURL = %q, want http://localhost:11434", p.baseURL)
	}
}

func TestNewOllama_WithTimeoutOption(t *testing.T) {
	p := NewOllama("http://example.com:11434", WithOllamaTimeout(5*time.Minute))
	if p.client.Timeout != 5*time.Minute {
		t.Errorf("timeout = %v, want 5m", p.client.Timeout)
	}
}

func TestNewOllama_WithTimeoutZeroDisablesClientTimeout(t *testing.T) {
	p := NewOllama("", WithOllamaTimeout(0))
	if p.client.Timeout != 0 {
		t.Errorf("timeout = %v, want 0 (caller relies on context)", p.client.Timeout)
	}
}

func TestNewOllama_EnvOverridesDefault(t *testing.T) {
	t.Setenv(envOllamaTimeout, "30s")
	p := NewOllama("")
	if p.client.Timeout != 30*time.Second {
		t.Errorf("env-overridden timeout = %v, want 30s", p.client.Timeout)
	}
}

func TestNewOllama_OptionWinsOverEnv(t *testing.T) {
	t.Setenv(envOllamaTimeout, "30s")
	p := NewOllama("", WithOllamaTimeout(5*time.Second))
	if p.client.Timeout != 5*time.Second {
		t.Errorf("option timeout = %v, want 5s (option must beat env)", p.client.Timeout)
	}
}

func TestNewOllama_InvalidEnvFallsBack(t *testing.T) {
	t.Setenv(envOllamaTimeout, "not-a-duration")
	p := NewOllama("")
	if p.client.Timeout != defaultOllamaTimeout {
		t.Errorf("invalid env timeout = %v, want default %v", p.client.Timeout, defaultOllamaTimeout)
	}
}

func TestNewOllama_WithHTTPClient(t *testing.T) {
	custom := &http.Client{Timeout: 7 * time.Second}
	p := NewOllama("", WithOllamaHTTPClient(custom))
	if p.client != custom {
		t.Error("expected provider client to be the injected one")
	}
	if p.client.Timeout != 7*time.Second {
		t.Errorf("client.Timeout = %v, want 7s", p.client.Timeout)
	}
}

func TestNewOllama_WithHTTPClientNilNoOp(t *testing.T) {
	p := NewOllama("", WithOllamaHTTPClient(nil))
	if p.client == nil {
		t.Error("nil client option must be a no-op, got nil client")
	}
}

func TestOllamaProvider_RespectsContextDeadline(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(800 * time.Millisecond):
			_, _ = w.Write([]byte(`{"message":{"role":"assistant","content":"late"},"done":true}`))
		case <-r.Context().Done():
		}
	}))
	defer srv.Close()

	// Disable the http.Client timeout entirely; the request context
	// at 100ms is the only deadline. This exercises the same path
	// callers like OSG (which already wraps each call in
	// context.WithTimeout(cfg.AI.Timeout)) want — kairos must not
	// silently override their deadline with its own client.Timeout.
	p := NewOllama(srv.URL, WithOllamaTimeout(0))
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := p.Chat(ctx, ChatRequest{
		Model:    "test",
		Messages: []Message{{Role: RoleUser, Content: "hi"}},
	})
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("expected a deadline error, got nil")
	}
	if elapsed > 600*time.Millisecond {
		t.Errorf("Chat took %v; expected to abort near the 100ms context deadline", elapsed)
	}
}
