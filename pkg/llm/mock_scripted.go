package llm

import (
	"context"
	"errors"
	"sync"
)

// ScriptedMockProvider is a mock provider that returns a pre-defined sequence of responses.
// Useful for testing multi-turn interactions (e.g. ReAct loop).
type ScriptedMockProvider struct {
	mu        sync.Mutex
	Responses []string
	Err       error
	// CallCount tracks how many times Chat has been called
	CallCount int
}

// NewScriptedMockProvider creates a new ScriptedMockProvider.
// The model argument is currently ignored by the mock but included for compatibility.
func NewScriptedMockProvider(model string, responses ...string) *ScriptedMockProvider {
	return &ScriptedMockProvider{
		Responses: responses,
	}
}

// Chat pops the next scripted response or returns the configured error.
func (s *ScriptedMockProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.CallCount++

	if s.Err != nil {
		return nil, s.Err
	}

	if len(s.Responses) == 0 {
		return nil, errors.New("scripted mock: no more responses available")
	}

	// Pop the first response
	content := s.Responses[0]
	s.Responses = s.Responses[1:]

	return &ChatResponse{
		Content: content,
		Usage: Usage{
			PromptTokens:     10,
			CompletionTokens: 10,
			TotalTokens:      20,
		},
	}, nil
}

// AddResponse appends a response to the queue.
func (s *ScriptedMockProvider) AddResponse(response string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Responses = append(s.Responses, response)
}

// PeekNext returns the next response to be returned, or empty string.
func (s *ScriptedMockProvider) PeekNext() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.Responses) == 0 {
		return ""
	}
	return s.Responses[0]
}
