package demo

import (
	"fmt"

	"github.com/google/uuid"
	a2av1 "github.com/jllopis/kairos/pkg/a2a/types"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	EventThinking       = "thinking"
	EventRetrievalStart = "retrieval.started"
	EventRetrievalDone  = "retrieval.done"
	EventToolStart      = "tool.started"
	EventToolProgress   = "tool.progress"
	EventToolDone       = "tool.done"
)

// NewTextMessage builds a minimal A2A message with a single text part.
func NewTextMessage(role a2av1.Role, text, contextID, taskID string) *a2av1.Message {
	return &a2av1.Message{
		MessageId: uuid.NewString(),
		ContextId: contextID,
		TaskId:    taskID,
		Role:      role,
		Parts: []*a2av1.Part{
			{Part: &a2av1.Part_Text{Text: text}},
		},
	}
}

// NewDataMessage builds a message with a JSON-compatible data payload.
func NewDataMessage(role a2av1.Role, data map[string]interface{}, contextID, taskID string) *a2av1.Message {
	payload, _ := structpb.NewStruct(data)
	return &a2av1.Message{
		MessageId: uuid.NewString(),
		ContextId: contextID,
		TaskId:    taskID,
		Role:      role,
		Parts: []*a2av1.Part{
			{Part: &a2av1.Part_Data{Data: &a2av1.DataPart{Data: payload}}},
		},
	}
}

// StatusEvent creates a stream response with a semantic event type in metadata.
func StatusEvent(taskID, contextID, eventType, message string, final bool) *a2av1.StreamResponse {
	return StatusEventWithState(taskID, contextID, eventType, message, a2av1.TaskState_TASK_STATE_WORKING, final)
}

// StatusEventWithState creates a stream response with a specific task state.
func StatusEventWithState(taskID, contextID, eventType, message string, state a2av1.TaskState, final bool) *a2av1.StreamResponse {
	metadata, _ := structpb.NewStruct(map[string]interface{}{
		"event_type": eventType,
	})
	statusMsg := NewTextMessage(a2av1.Role_ROLE_AGENT, message, contextID, taskID)
	status := &a2av1.TaskStatus{
		State:     state,
		Message:   statusMsg,
		Timestamp: timestamppb.Now(),
	}
	update := &a2av1.TaskStatusUpdateEvent{
		TaskId:    taskID,
		ContextId: contextID,
		Status:    status,
		Final:     final,
		Metadata:  metadata,
	}
	return &a2av1.StreamResponse{Payload: &a2av1.StreamResponse_StatusUpdate{StatusUpdate: update}}
}

// FormatTable builds a text table for quick CLI rendering.
func FormatTable(headers []string, rows [][]string) string {
	out := ""
	if len(headers) > 0 {
		out += fmt.Sprintf("%s\n", joinWithTabs(headers))
	}
	for _, row := range rows {
		out += fmt.Sprintf("%s\n", joinWithTabs(row))
	}
	return out
}

func joinWithTabs(parts []string) string {
	return fmt.Sprintf("%s", join(parts, "\t"))
}

func join(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for i := 1; i < len(parts); i++ {
		out += sep + parts[i]
	}
	return out
}
