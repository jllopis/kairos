package llm

import (
	"context"
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
