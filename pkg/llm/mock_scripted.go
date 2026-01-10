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

// AddResponse adds a response to the queue
func (s *ScriptedMockProvider) AddResponse(response string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Responses = append(s.Responses, response)
}

// LastResponse returns the last response that will be returned, or empty string
func (s *ScriptedMockProvider) PeekNext() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.Responses) == 0 {
		return ""
	}
	return s.Responses[0]
}
