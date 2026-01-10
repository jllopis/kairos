package runtime

import (
	"context"
	"errors"

	"github.com/jllopis/kairos/pkg/core"
)

// Runtime defines the minimal lifecycle for executing agents.
type Runtime interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Run(ctx context.Context, agent core.Agent, input any) (any, error)
}

// LocalRuntime is a simple in-process runtime.
type LocalRuntime struct {
	started bool
}

// NewLocal creates a new LocalRuntime instance.
func NewLocal() *LocalRuntime {
	return &LocalRuntime{}
}

// Start marks the runtime as ready.
func (r *LocalRuntime) Start(_ context.Context) error {
	r.started = true
	return nil
}

// Stop marks the runtime as stopped.
func (r *LocalRuntime) Stop(_ context.Context) error {
	r.started = false
	return nil
}

// Run executes an agent using the provided context.
func (r *LocalRuntime) Run(ctx context.Context, agent core.Agent, input any) (any, error) {
	if !r.started {
		return nil, errors.New("runtime not started")
	}
	ctx, _ = core.EnsureRunID(ctx)
	return agent.Run(ctx, input)
}
