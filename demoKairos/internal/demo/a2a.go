package demo

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	a2av1 "github.com/jllopis/kairos/pkg/a2a/types"
	"github.com/jllopis/kairos/pkg/core"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
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
	payload, err := structpb.NewStruct(normalizeStructMap(data))
	if err != nil {
		payload, _ = structpb.NewStruct(map[string]interface{}{
			"error": err.Error(),
		})
	}
	if payload == nil {
		payload = &structpb.Struct{}
	}
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
func StatusEvent(taskID, contextID string, eventType core.EventType, message string, payload map[string]any, final bool) *a2av1.StreamResponse {
	return StatusEventWithState(taskID, contextID, eventType, message, payload, a2av1.TaskState_TASK_STATE_WORKING, final)
}

// StatusEventWithState creates a stream response with a specific task state.
func StatusEventWithState(taskID, contextID string, eventType core.EventType, message string, payload map[string]any, state a2av1.TaskState, final bool) *a2av1.StreamResponse {
	meta := map[string]interface{}{
		"event_type": string(eventType),
	}
	if payload != nil {
		meta["payload"] = payload
	}
	metadata, _ := structpb.NewStruct(normalizeStructMap(meta))
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

func normalizeStructMap(values map[string]interface{}) map[string]interface{} {
	if values == nil {
		return map[string]interface{}{}
	}
	out := make(map[string]interface{}, len(values))
	for key, value := range values {
		out[key] = normalizeStructValue(value)
	}
	return out
}

func normalizeStructValue(value interface{}) interface{} {
	switch typed := value.(type) {
	case nil:
		return nil
	case string:
		return typed
	case bool:
		return typed
	case int:
		return float64(typed)
	case int8:
		return float64(typed)
	case int16:
		return float64(typed)
	case int32:
		return float64(typed)
	case int64:
		return float64(typed)
	case uint:
		return float64(typed)
	case uint8:
		return float64(typed)
	case uint16:
		return float64(typed)
	case uint32:
		return float64(typed)
	case uint64:
		return float64(typed)
	case float32:
		return float64(typed)
	case float64:
		return typed
	case time.Time:
		return typed.Format(time.RFC3339Nano)
	case []string:
		out := make([]interface{}, 0, len(typed))
		for _, item := range typed {
			out = append(out, item)
		}
		return out
	case []interface{}:
		out := make([]interface{}, 0, len(typed))
		for _, item := range typed {
			out = append(out, normalizeStructValue(item))
		}
		return out
	case map[string]interface{}:
		return normalizeStructMap(typed)
	case map[string]string:
		out := make(map[string]interface{}, len(typed))
		for key, item := range typed {
			out[key] = item
		}
		return out
	default:
		return fmt.Sprint(value)
	}
}
