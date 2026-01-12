// Package runtime provides agent execution environments.
package runtime

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jllopis/kairos/pkg/core"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
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
	tracer  trace.Tracer
}

// NewLocal creates a new LocalRuntime instance.
func NewLocal() *LocalRuntime {
	return &LocalRuntime{}
}

// Start marks the runtime as ready.
func (r *LocalRuntime) Start(_ context.Context) error {
	r.started = true
	if r.tracer == nil {
		r.tracer = otel.Tracer("kairos/runtime")
	}
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
	ctx, runID := core.EnsureRunID(ctx)
	if r.tracer == nil {
		r.tracer = otel.Tracer("kairos/runtime")
	}
	log := slog.Default()
	log.Info("runtime.run.start",
		slog.String("agent_id", agent.ID()),
		slog.String("run_id", runID),
	)
	ctx, span := r.tracer.Start(ctx, "Runtime.Run", trace.WithAttributes(
		attribute.String("agent.id", agent.ID()),
	))
	defer span.End()
	traceID, spanID := traceIDs(span)
	result, err := agent.Run(ctx, input)
	if err != nil {
		log.Error("runtime.run.error",
			slog.String("agent_id", agent.ID()),
			slog.String("run_id", runID),
			slog.String("trace_id", traceID),
			slog.String("span_id", spanID),
			slog.String("error", err.Error()),
		)
		return nil, err
	}
	log.Info("runtime.run.complete",
		slog.String("agent_id", agent.ID()),
		slog.String("run_id", runID),
		slog.String("trace_id", traceID),
		slog.String("span_id", spanID),
	)
	return result, nil
}

func traceIDs(span trace.Span) (string, string) {
	sc := span.SpanContext()
	return sc.TraceID().String(), sc.SpanID().String()
}
