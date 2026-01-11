package server

import (
	a2av1 "github.com/jllopis/kairos/pkg/a2a/types"
	"github.com/jllopis/kairos/pkg/core"
)

// HandlerOption customizes the SimpleHandler wiring.
type HandlerOption func(*SimpleHandler)

// WithStore overrides the task store.
func WithStore(store TaskStore) HandlerOption {
	return func(h *SimpleHandler) {
		if store != nil {
			h.Store = store
		}
	}
}

// WithExecutor overrides the executor.
func WithExecutor(exec Executor) HandlerOption {
	return func(h *SimpleHandler) {
		if exec != nil {
			h.Executor = exec
		}
	}
}

// WithAgentCard configures the handler AgentCard (used for GetExtendedAgentCard).
func WithAgentCard(card *a2av1.AgentCard) HandlerOption {
	return func(h *SimpleHandler) {
		if card != nil {
			h.AgentCard = card
		}
	}
}

// NewAgentHandler wires a SimpleHandler to a Kairos agent.
func NewAgentHandler(agent core.Agent, opts ...HandlerOption) *SimpleHandler {
	handler := &SimpleHandler{
		Store:    NewMemoryTaskStore(),
		Executor: &AgentExecutor{Agent: agent},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(handler)
		}
	}
	return handler
}

// NewAgentService creates a gRPC A2A service wired to a Kairos agent.
func NewAgentService(agent core.Agent, opts ...HandlerOption) *Service {
	return New(NewAgentHandler(agent, opts...))
}
