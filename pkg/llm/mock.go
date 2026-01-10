package llm

import (
	"context"
	"fmt"
)

// MockProvider is a testing implementation of Provider.
type MockProvider struct {
	Response string
	Err      error
	ChatFunc func(ctx context.Context, req ChatRequest) (*ChatResponse, error)
}

func (m *MockProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if m.ChatFunc != nil {
		return m.ChatFunc(ctx, req)
	}
	if m.Err != nil {
		return nil, m.Err
	}
	return &ChatResponse{
		Content: m.Response,
		Usage: Usage{
			PromptTokens:     10,
			CompletionTokens: 10,
			TotalTokens:      20,
		},
	}, nil
}

// FailingMockProvider always fails.
type FailingMockProvider struct {
	Err error
}

func (f *FailingMockProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if f.Err == nil {
		return nil, fmt.Errorf("mock error")
	}
	return nil, f.Err
}
