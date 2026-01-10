package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jllopis/kairos/pkg/llm"
	"github.com/mark3labs/mcp-go/mcp"
)

type stubCaller struct {
	lastName string
	lastArgs map[string]interface{}
	result   *mcp.CallToolResult
	err      error
}

func (s *stubCaller) CallTool(_ context.Context, name string, args map[string]interface{}) (*mcp.CallToolResult, error) {
	s.lastName = name
	s.lastArgs = args
	return s.result, s.err
}

func TestToolAdapter_Call_MapsStringInput(t *testing.T) {
	tool := mcp.Tool{
		Name: "echo",
		InputSchema: mcp.ToolInputSchema{
			Type:     "object",
			Required: []string{"input"},
		},
	}

	caller := &stubCaller{
		result: &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "ok"}},
		},
	}

	adapter, err := NewToolAdapter(tool, caller)
	if err != nil {
		t.Fatalf("NewToolAdapter error: %v", err)
	}

	output, err := adapter.Call(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Call error: %v", err)
	}

	if output != "ok" {
		t.Fatalf("Expected output 'ok', got %v", output)
	}

	if caller.lastName != "echo" {
		t.Fatalf("Expected tool name 'echo', got %q", caller.lastName)
	}

	if caller.lastArgs["input"] != "hello" {
		t.Fatalf("Expected input arg to be 'hello', got %v", caller.lastArgs["input"])
	}
}

func TestToolAdapter_Call_ParsesJSONInput(t *testing.T) {
	tool := mcp.Tool{
		Name: "sum",
		InputSchema: mcp.ToolInputSchema{
			Type:     "object",
			Required: []string{"a", "b"},
		},
	}

	caller := &stubCaller{
		result: &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "3"}},
		},
	}

	adapter, err := NewToolAdapter(tool, caller)
	if err != nil {
		t.Fatalf("NewToolAdapter error: %v", err)
	}

	output, err := adapter.Call(context.Background(), `{"a":1,"b":2}`)
	if err != nil {
		t.Fatalf("Call error: %v", err)
	}

	if output != "3" {
		t.Fatalf("Expected output '3', got %v", output)
	}

	if caller.lastArgs["a"] != float64(1) || caller.lastArgs["b"] != float64(2) {
		t.Fatalf("Expected args a=1 b=2, got %v", caller.lastArgs)
	}
}

func TestToolAdapter_Call_ValidatesRequiredArgs(t *testing.T) {
	tool := mcp.Tool{
		Name: "needs-foo",
		InputSchema: mcp.ToolInputSchema{
			Type:     "object",
			Required: []string{"foo"},
		},
	}

	caller := &stubCaller{
		result: &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "ok"}},
		},
	}

	adapter, err := NewToolAdapter(tool, caller)
	if err != nil {
		t.Fatalf("NewToolAdapter error: %v", err)
	}

	_, err = adapter.Call(context.Background(), map[string]interface{}{"bar": "baz"})
	if err == nil || !strings.Contains(err.Error(), "missing required field") {
		t.Fatalf("Expected missing required field error, got %v", err)
	}
}

func TestToolAdapter_Call_ReturnsStructuredContent(t *testing.T) {
	tool := mcp.Tool{Name: "structured"}
	caller := &stubCaller{
		result: &mcp.CallToolResult{
			StructuredContent: map[string]interface{}{"ok": true},
		},
	}

	adapter, err := NewToolAdapter(tool, caller)
	if err != nil {
		t.Fatalf("NewToolAdapter error: %v", err)
	}

	output, err := adapter.Call(context.Background(), nil)
	if err != nil {
		t.Fatalf("Call error: %v", err)
	}

	payload, ok := output.(map[string]interface{})
	if !ok || payload["ok"] != true {
		t.Fatalf("Expected structured payload, got %v", output)
	}
}

func TestToolDefinition_UsesRawSchema(t *testing.T) {
	raw := json.RawMessage(`{"type":"object","properties":{"q":{"type":"string"}}}`)
	tool := mcp.Tool{
		Name:           "search",
		Description:    "Search tool",
		RawInputSchema: raw,
	}

	def := ToolDefinition(tool)
	if def.Type != llm.ToolTypeFunction {
		t.Fatalf("Expected function tool, got %v", def.Type)
	}

	rawParams, ok := def.Function.Parameters.(json.RawMessage)
	if !ok {
		t.Fatalf("Expected raw schema parameters, got %T", def.Function.Parameters)
	}

	if string(rawParams) != string(raw) {
		t.Fatalf("Unexpected raw schema %s", string(rawParams))
	}
}
