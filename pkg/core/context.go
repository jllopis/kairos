// Package core defines shared interfaces and context helpers for Kairos.
package core

import (
	"context"
	"crypto/rand"
	"encoding/hex"
)

type runIDKey struct{}
type memoryKey struct{}
type taskKey struct{}
type sessionIDKey struct{}

// WithRunID attaches a run id to the context.
func WithRunID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, runIDKey{}, id)
}

// RunID returns the run id if present.
func RunID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(runIDKey{}).(string)
	return id, ok
}

// EnsureRunID ensures a run id exists in the context.
func EnsureRunID(ctx context.Context) (context.Context, string) {
	if id, ok := RunID(ctx); ok {
		return ctx, id
	}
	id := newRunID()
	return WithRunID(ctx, id), id
}

// WithMemory attaches a memory backend to the context.
func WithMemory(ctx context.Context, mem Memory) context.Context {
	return context.WithValue(ctx, memoryKey{}, mem)
}

// MemoryFromContext returns the memory backend if present.
func MemoryFromContext(ctx context.Context) (Memory, bool) {
	mem, ok := ctx.Value(memoryKey{}).(Memory)
	return mem, ok
}

// WithTask attaches a task to the context.
func WithTask(ctx context.Context, task *Task) context.Context {
	return context.WithValue(ctx, taskKey{}, task)
}

// TaskFromContext returns the task if present.
func TaskFromContext(ctx context.Context) (*Task, bool) {
	task, ok := ctx.Value(taskKey{}).(*Task)
	return task, ok
}

// WithSessionID attaches a conversation session id to the context.
func WithSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, sessionIDKey{}, sessionID)
}

// SessionID returns the session id if present.
func SessionID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(sessionIDKey{}).(string)
	return id, ok
}

// EnsureSessionID ensures a session id exists in the context.
func EnsureSessionID(ctx context.Context) (context.Context, string) {
	if id, ok := SessionID(ctx); ok && id != "" {
		return ctx, id
	}
	id := newSessionID()
	return WithSessionID(ctx, id), id
}

func newRunID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "run-unknown"
	}
	return "run-" + hex.EncodeToString(buf)
}

func newSessionID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "session-unknown"
	}
	return "session-" + hex.EncodeToString(buf)
}
