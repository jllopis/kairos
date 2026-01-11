package server

import (
	"context"
	"encoding/json"
	"fmt"

	a2av1 "github.com/jllopis/kairos/pkg/a2a/types"
	"github.com/jllopis/kairos/pkg/core"
)

// AgentExecutor maps A2A messages to a Kairos agent execution.
type AgentExecutor struct {
	Agent core.Agent
}

// Run executes the agent with input extracted from the A2A message.
func (e *AgentExecutor) Run(ctx context.Context, message *a2av1.Message) (any, []*a2av1.Artifact, error) {
	if e.Agent == nil {
		return nil, nil, errMissingAgent()
	}

	input, err := messageToInput(message)
	if err != nil {
		return nil, nil, err
	}

	output, err := e.Agent.Run(ctx, input)
	if err != nil {
		return nil, nil, err
	}

	return output, nil, nil
}

func messageToInput(message *a2av1.Message) (any, error) {
	if text := ExtractText(message); text != "" {
		return text, nil
	}

	if data := ExtractData(message); data != nil {
		encoded, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		return string(encoded), nil
	}

	return "", nil
}

func errMissingAgent() error {
	return fmt.Errorf("agent executor requires a non-nil agent")
}
