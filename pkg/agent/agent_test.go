package agent_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/jllopis/kairos/pkg/agent"
	"github.com/jllopis/kairos/pkg/core"
	"github.com/jllopis/kairos/pkg/llm"
)

// MockTool implements core.Tool for testing
type MockTool struct {
	NameVal string
}

func (m *MockTool) Name() string { return m.NameVal }
func (m *MockTool) Call(ctx context.Context, input any) (any, error) {
	return fmt.Sprintf("Result from %s with %v", m.NameVal, input), nil
}

func TestAgent_ReActLoop(t *testing.T) {
	ctx := context.Background()

	// Setup Toolkit
	tool := &MockTool{NameVal: "Calculator"}

	// Setup Scripted Mock LLM
	// Scenario:
	// 1. User: "What is 10 + 5?"
	// 2. LLM: "Thought: need math. Action: Calculator\nAction Input: 10 + 5"
	// 3. Agent: Executes Calculator -> "Result from Calculator with 10 + 5"
	// 4. LLM: "Final Answer: 15"
	mockLLM := &llm.ScriptedMockProvider{}
	mockLLM.AddResponse("Thought: need math. Action: Calculator\nAction Input: 10 + 5")
	mockLLM.AddResponse("Final Answer: 15")

	// Create Agent
	a, err := agent.New("test-agent", mockLLM,
		agent.WithTools([]core.Tool{tool}),
	)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Run Agent
	result, err := a.Run(ctx, "What is 10 + 5?")
	if err != nil {
		t.Fatalf("Agent.Run failing: %v", err)
	}

	// Assertions
	if result != "15" {
		t.Errorf("Expected result '15', got '%v'", result)
	}

	if mockLLM.CallCount != 2 {
		t.Errorf("Expected 2 LLM calls, got %d", mockLLM.CallCount)
	}
}

func TestAgent_SingleTurn(t *testing.T) {
	ctx := context.Background()
	mockLLM := &llm.ScriptedMockProvider{}
	mockLLM.AddResponse("Just a chat response.")

	a, err := agent.New("chat-agent", mockLLM)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	result, err := a.Run(ctx, "Hello")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if result != "Just a chat response." {
		t.Errorf("Unexpected result: %v", result)
	}
}
