package core

import (
	"context"
	"time"
)

// EventType identifies a semantic event emitted by agents or runtimes.
type EventType string

const (
	EventAgentThinking      EventType = "agent.thinking"
	EventAgentTaskStarted   EventType = "agent.task.started"
	EventAgentTaskCompleted EventType = "agent.task.completed"
	EventAgentDelegation    EventType = "agent.delegation"
	EventAgentError         EventType = "agent.error"
)

// Event captures a semantic streaming/logging event.
type Event struct {
	Type      EventType
	Agent     string
	TaskID    string
	Timestamp time.Time
	Payload   map[string]any
}

// EventEmitter receives semantic events.
type EventEmitter interface {
	Emit(ctx context.Context, event Event)
}

// NoopEventEmitter is a default no-op implementation.
type NoopEventEmitter struct{}

// Emit implements EventEmitter.
func (NoopEventEmitter) Emit(_ context.Context, _ Event) {}

// NewEvent builds a default event with timestamp.
func NewEvent(eventType EventType, agent string, taskID string, payload map[string]any) Event {
	return Event{
		Type:      eventType,
		Agent:     agent,
		TaskID:    taskID,
		Timestamp: time.Now().UTC(),
		Payload:   payload,
	}
}
