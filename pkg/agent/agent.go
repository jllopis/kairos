package agent

import (
	"context"
	"errors"

	"github.com/jllopis/kairos/pkg/core"
)

// Handler executes the agent's core behavior.
type Handler func(ctx context.Context, input any) (any, error)

// Agent is a simple implementation of the core.Agent interface.
type Agent struct {
	id      string
	role    string
	skills  []core.Skill
	memory  core.Memory
	handler Handler
}

var ErrMissingHandler = errors.New("agent handler is required")

// Option configures an Agent instance.
type Option func(*Agent) error

// New creates a new Agent with a required id and options.
func New(id string, opts ...Option) (*Agent, error) {
	a := &Agent{id: id}
	for _, opt := range opts {
		if err := opt(a); err != nil {
			return nil, err
		}
	}
	if a.id == "" {
		return nil, errors.New("agent id is required")
	}
	if a.handler == nil {
		return nil, ErrMissingHandler
	}
	return a, nil
}

// WithRole sets the agent role.
func WithRole(role string) Option {
	return func(a *Agent) error {
		a.role = role
		return nil
	}
}

// WithSkills assigns semantic skills to the agent.
func WithSkills(skills []core.Skill) Option {
	return func(a *Agent) error {
		a.skills = append([]core.Skill(nil), skills...)
		return nil
	}
}

// WithMemory attaches a memory backend to the agent.
func WithMemory(memory core.Memory) Option {
	return func(a *Agent) error {
		a.memory = memory
		return nil
	}
}

// WithHandler sets the agent handler.
func WithHandler(handler Handler) Option {
	return func(a *Agent) error {
		a.handler = handler
		return nil
	}
}

// ID returns the agent identifier.
func (a *Agent) ID() string { return a.id }

// Role returns the agent role.
func (a *Agent) Role() string { return a.role }

// Skills returns the agent skills.
func (a *Agent) Skills() []core.Skill {
	return append([]core.Skill(nil), a.skills...)
}

// Memory returns the attached memory backend, if any.
func (a *Agent) Memory() core.Memory { return a.memory }

// Run executes the agent handler.
func (a *Agent) Run(ctx context.Context, input any) (any, error) {
	if a.handler == nil {
		return nil, ErrMissingHandler
	}
	if a.memory != nil {
		ctx = core.WithMemory(ctx, a.memory)
	}
	return a.handler(ctx, input)
}
