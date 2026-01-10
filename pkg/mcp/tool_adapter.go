package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jllopis/kairos/pkg/core"
	"github.com/jllopis/kairos/pkg/llm"
	"github.com/mark3labs/mcp-go/mcp"
)

// ToolCaller abstracts MCP tool execution for adapters.
type ToolCaller interface {
	CallTool(ctx context.Context, name string, args map[string]interface{}) (*mcp.CallToolResult, error)
}

// ToolAdapter wraps an MCP tool to satisfy core.Tool.
type ToolAdapter struct {
	tool   mcp.Tool
	caller ToolCaller
}

// NewToolAdapter builds a core.Tool backed by an MCP tool definition and caller.
func NewToolAdapter(tool mcp.Tool, caller ToolCaller) (*ToolAdapter, error) {
	if tool.Name == "" {
		return nil, errors.New("mcp tool name is required")
	}
	if caller == nil {
		return nil, errors.New("tool caller is required")
	}
	return &ToolAdapter{
		tool:   tool,
		caller: caller,
	}, nil
}

// Name returns the MCP tool name.
func (t *ToolAdapter) Name() string {
	return t.tool.Name
}

// ToolDefinition returns an LLM function definition for this tool.
func (t *ToolAdapter) ToolDefinition() llm.Tool {
	return ToolDefinition(t.tool)
}

// Call invokes the MCP tool with normalized arguments.
func (t *ToolAdapter) Call(ctx context.Context, input any) (any, error) {
	args, err := normalizeToolArgs(input)
	if err != nil {
		return nil, err
	}

	if raw, ok := input.(string); ok {
		trimmed := strings.TrimSpace(raw)
		if trimmed != "" {
			if _, hasURL := args["url"]; !hasURL && requiresField(t.tool, "url") {
				args = map[string]interface{}{"url": trimmed}
			}
		}
	}

	if err := validateRequiredArgs(t.tool, args); err != nil {
		return nil, err
	}

	result, err := t.caller.CallTool(ctx, t.tool.Name, args)
	if err != nil {
		return nil, err
	}

	return toolResultToOutput(result)
}

// ToolDefinition converts an MCP tool into an LLM function tool definition.
func ToolDefinition(tool mcp.Tool) llm.Tool {
	var params any = tool.InputSchema
	if tool.RawInputSchema != nil {
		params = tool.RawInputSchema
	}
	return llm.Tool{
		Type: llm.ToolTypeFunction,
		Function: llm.FunctionDef{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  params,
		},
	}
}

// ToolDefinitions converts MCP tools to LLM function tool definitions.
func ToolDefinitions(tools []mcp.Tool) []llm.Tool {
	defs := make([]llm.Tool, 0, len(tools))
	for _, tool := range tools {
		defs = append(defs, ToolDefinition(tool))
	}
	return defs
}

func normalizeToolArgs(input any) (map[string]interface{}, error) {
	switch value := input.(type) {
	case nil:
		return map[string]interface{}{}, nil
	case map[string]interface{}:
		return value, nil
	case json.RawMessage:
		var decoded map[string]interface{}
		if err := json.Unmarshal(value, &decoded); err != nil {
			return nil, fmt.Errorf("mcp tool args: invalid JSON: %w", err)
		}
		return decoded, nil
	case []byte:
		var decoded map[string]interface{}
		if err := json.Unmarshal(value, &decoded); err != nil {
			return nil, fmt.Errorf("mcp tool args: invalid JSON: %w", err)
		}
		return decoded, nil
	case string:
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return map[string]interface{}{}, nil
		}
		if strings.HasPrefix(trimmed, "{") {
			var decoded map[string]interface{}
			if err := json.Unmarshal([]byte(trimmed), &decoded); err == nil {
				return decoded, nil
			}
		}
		return map[string]interface{}{"input": value}, nil
	default:
		encoded, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("mcp tool args: unsupported type %T", input)
		}
		var decoded map[string]interface{}
		if err := json.Unmarshal(encoded, &decoded); err != nil {
			return nil, fmt.Errorf("mcp tool args: invalid JSON after marshal: %w", err)
		}
		return decoded, nil
	}
}

func validateRequiredArgs(tool mcp.Tool, args map[string]interface{}) error {
	schema := tool.InputSchema
	if schema.Type != "" && schema.Type != "object" {
		return nil
	}
	for _, key := range schema.Required {
		if _, ok := args[key]; !ok {
			return fmt.Errorf("mcp tool args: missing required field %q", key)
		}
	}
	return nil
}

func requiresField(tool mcp.Tool, name string) bool {
	schema := tool.InputSchema
	for _, key := range schema.Required {
		if key == name {
			return true
		}
	}
	return false
}

func toolResultToOutput(result *mcp.CallToolResult) (any, error) {
	if result == nil {
		return nil, errors.New("mcp tool result is nil")
	}

	if result.IsError {
		return nil, fmt.Errorf("mcp tool returned error: %s", extractTextContent(result.Content))
	}

	if result.StructuredContent != nil {
		return result.StructuredContent, nil
	}

	if text := extractTextContent(result.Content); text != "" {
		return text, nil
	}

	return result, nil
}

func extractTextContent(items []mcp.Content) string {
	if len(items) == 0 {
		return ""
	}
	var parts []string
	for _, item := range items {
		switch content := item.(type) {
		case mcp.TextContent:
			parts = append(parts, content.Text)
		case *mcp.TextContent:
			parts = append(parts, content.Text)
		}
	}
	return strings.Join(parts, "\n")
}

var _ core.Tool = (*ToolAdapter)(nil)
