package agent_test

import (
	"context"
	"fmt"
	"sync"
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
func (m *MockTool) ToolDefinition() llm.Tool {
	return llm.Tool{
		Type: llm.ToolTypeFunction,
		Function: llm.FunctionDef{
			Name: m.NameVal,
			Parameters: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
	}
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
	ToolName  string
	ToolArgs  string
	Final     string
}

func (p *toolCallProvider) Chat(_ context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	p.CallCount++
	p.LastReq = req
	if p.CallCount == 1 {
		toolName := p.ToolName
		if toolName == "" {
			toolName = "search"
		}
		toolArgs := p.ToolArgs
		if toolArgs == "" {
			toolArgs = `{"query":"hello"}`
		}
		return &llm.ChatResponse{
			Content: "",
			ToolCalls: []llm.ToolCall{
				{
					ID:   "call-1",
					Type: llm.ToolTypeFunction,
					Function: llm.FunctionCall{
						Name:      toolName,
						Arguments: toolArgs,
					},
				},
			},
		}, nil
	}
	final := p.Final
	if final == "" {
		final = "Final Answer: done"
	}
	return &llm.ChatResponse{Content: final}, nil
}

type eventCollector struct {
	mu     sync.Mutex
	events []core.Event
}

func (c *eventCollector) Emit(_ context.Context, event core.Event) {
	c.mu.Lock()
	c.events = append(c.events, event)
	c.mu.Unlock()
}

func (c *eventCollector) types() []core.EventType {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]core.EventType, 0, len(c.events))
	for _, event := range c.events {
		out = append(out, event.Type)
	}
	return out
}

func TestAgent_ReActLoop(t *testing.T) {
	ctx := context.Background()

	// Setup Toolkit
	tool := &MockTool{NameVal: "Calculator"}

	// Setup ToolCall LLM
	toolCallLLM := &toolCallProvider{
		ToolName: "Calculator",
		ToolArgs: `{"input":"10 + 5"}`,
		Final:    "Final Answer: 15",
	}

	// Create Agent
	a, err := agent.New("test-agent", toolCallLLM,
		agent.WithTools([]core.Tool{tool}),
		agent.WithDisableActionFallback(true),
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

	if toolCallLLM.CallCount != 2 {
		t.Errorf("Expected 2 LLM calls, got %d", toolCallLLM.CallCount)
	}
}

func TestAgent_EmitsSemanticEvents(t *testing.T) {
	ctx := context.Background()
	emitter := &eventCollector{}
	llmProvider := &llm.MockProvider{Response: "Final Answer: listo"}

	a, err := agent.New("event-agent", llmProvider,
		agent.WithRole("Tester"),
		agent.WithEventEmitter(emitter),
	)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}
	if _, err := a.Run(ctx, "ping"); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	types := emitter.types()
	foundStarted := false
	foundCompleted := false
	for _, tpe := range types {
		switch tpe {
		case core.EventAgentTaskStarted:
			foundStarted = true
		case core.EventAgentTaskCompleted:
			foundCompleted = true
		}
	}
	if !foundStarted {
		t.Fatalf("expected task started event")
	}
	if !foundCompleted {
		t.Fatalf("expected task completed event")
	}
}

func TestAgent_UpdatesTaskFromContext(t *testing.T) {
	ctx := context.Background()
	task := core.NewTask("ping", "")
	ctx = core.WithTask(ctx, task)
	llmProvider := &llm.MockProvider{Response: "Final Answer: ok"}

	a, err := agent.New("task-agent", llmProvider)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}
	if _, err := a.Run(ctx, "ping"); err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if task.Status != core.TaskStatusCompleted {
		t.Fatalf("expected task completed, got %s", task.Status)
	}
	if task.AssignedTo != "task-agent" {
		t.Fatalf("expected assigned_to set")
	}
	if task.Result == nil {
		t.Fatalf("expected task result")
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
	provider := &toolCallProvider{
		ToolName: "search",
		ToolArgs: `{"query":"hello"}`,
		Final:    "Final Answer: done",
	}

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
