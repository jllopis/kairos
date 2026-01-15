package core

import "context"

// Skill represents a semantic capability as defined by AgentSkills.
type Skill struct {
	Name          string
	Description   string
	License       string
	Compatibility string
	Metadata      map[string]string
	AllowedTools  []string
	Body          string
}

// Tool is a concrete implementation, typically backed by MCP.
type Tool interface {
	Name() string
	Call(ctx context.Context, input any) (any, error)
}

// Memory stores and retrieves contextual data for agents.
type Memory interface {
	Store(ctx context.Context, data any) error
	Retrieve(ctx context.Context, query any) (any, error)
}

// Plan represents a planning artifact (graph or emergent).
type Plan interface {
	ID() string
}

// Agent is the minimal executable unit of the runtime.
type Agent interface {
	ID() string
	Role() string
	Skills() []Skill
	Memory() Memory
	Run(ctx context.Context, input any) (any, error)
}
