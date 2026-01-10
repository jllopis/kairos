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

type toolWithDefinition struct {
	NameVal  string
	LastArgs any
}

func (t *toolWithDefinition) Name() string { return t.NameVal }
func (t *toolWithDefinition) Call(ctx context.Context, input any) (any, error) {
	t.LastArgs = input
	return fmt.Sprintf("ok:%s", t.NameVal), nil
}
func (t *toolWithDefinition) ToolDefinition() llm.Tool {
	return llm.Tool{
		Type: llm.ToolTypeFunction,
		Function: llm.FunctionDef{
			Name: t.NameVal,
			Parameters: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{"query": map[string]interface{}{"type": "string"}},
			},
		},
	}
}

type toolCallProvider struct {
	CallCount int
	LastReq   llm.ChatRequest
}

func (p *toolCallProvider) Chat(_ context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	p.CallCount++
	p.LastReq = req
	if p.CallCount == 1 {
		return &llm.ChatResponse{
			Content: "",
			ToolCalls: []llm.ToolCall{
				{
					ID:   "call-1",
					Type: llm.ToolTypeFunction,
					Function: llm.FunctionCall{
						Name:      "search",
						Arguments: `{"query":"hello"}`,
					},
				},
			},
		}, nil
	}
	return &llm.ChatResponse{Content: "Final Answer: done"}, nil
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

func TestAgent_ToolCallsStructured(t *testing.T) {
	ctx := context.Background()

	tool := &toolWithDefinition{NameVal: "search"}
	provider := &toolCallProvider{}

	a, err := agent.New("tool-call-agent", provider, agent.WithTools([]core.Tool{tool}))
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	result, err := a.Run(ctx, "Use the tool")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if result != "done" {
		t.Fatalf("Expected result 'done', got '%v'", result)
	}
	if provider.CallCount != 2 {
		t.Fatalf("Expected 2 LLM calls, got %d", provider.CallCount)
	}
	if len(provider.LastReq.Tools) == 0 {
		t.Fatalf("Expected tool definitions to be passed to LLM")
	}
	args, ok := tool.LastArgs.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected tool args to be map, got %T", tool.LastArgs)
	}
	if args["query"] != "hello" {
		t.Fatalf("Expected query arg 'hello', got %v", args["query"])
	}
}

type captureModelProvider struct {
	LastModel string
}

func (c *captureModelProvider) Chat(_ context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	c.LastModel = req.Model
	return &llm.ChatResponse{Content: "ok"}, nil
}

func TestAgent_WithModel(t *testing.T) {
	ctx := context.Background()
	capture := &captureModelProvider{}

	a, err := agent.New("model-agent", capture, agent.WithModel("kairos-test-model"))
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	if _, err := a.Run(ctx, "Hello"); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if capture.LastModel != "kairos-test-model" {
		t.Fatalf("Expected model 'kairos-test-model', got '%s'", capture.LastModel)
	}
}
