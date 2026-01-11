package server

import (
	"fmt"

	"github.com/google/uuid"
	a2av1 "github.com/jllopis/kairos/pkg/a2a/types"
	"google.golang.org/protobuf/types/known/structpb"
)

// ResponseMessage builds a message from an executor output.
func ResponseMessage(output any, contextID, taskID string) *a2av1.Message {
	if msg, ok := output.(*a2av1.Message); ok {
		return normalizeMessage(msg, contextID, taskID, a2av1.Role_ROLE_AGENT)
	}

	text := fmt.Sprint(output)
	part := &a2av1.Part{Part: &a2av1.Part_Text{Text: text}}
	return &a2av1.Message{
		MessageId: uuid.NewString(),
		ContextId: contextID,
		TaskId:    taskID,
		Role:      a2av1.Role_ROLE_AGENT,
		Parts:     []*a2av1.Part{part},
	}
}

// ValidateMessage ensures required fields are present.
func ValidateMessage(message *a2av1.Message) error {
	if message == nil {
		return fmt.Errorf("message is nil")
	}
	if message.MessageId == "" {
		return fmt.Errorf("message_id is required")
	}
	if len(message.Parts) == 0 {
		return fmt.Errorf("message parts are required")
	}
	return nil
}

func normalizeMessage(message *a2av1.Message, contextID, taskID string, role a2av1.Role) *a2av1.Message {
	if message.MessageId == "" {
		message.MessageId = uuid.NewString()
	}
	if contextID != "" {
		message.ContextId = contextID
	}
	if taskID != "" {
		message.TaskId = taskID
	}
	if message.Role == a2av1.Role_ROLE_UNSPECIFIED {
		message.Role = role
	}
	return message
}

// ExtractText returns concatenated text parts.
func ExtractText(message *a2av1.Message) string {
	if message == nil {
		return ""
	}
	var out string
	for _, part := range message.Parts {
		if part == nil {
			continue
		}
		if text := part.GetText(); text != "" {
			out += text
		}
	}
	return out
}

func structFromMap(data map[string]interface{}) *structpb.Struct {
	if len(data) == 0 {
		return nil
	}
	payload, err := structpb.NewStruct(data)
	if err != nil {
		return nil
	}
	return payload
}
