package llm

import "context"

// Role represents the role of a message sender.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// ToolType represents the type of tool.
type ToolType string

const (
	ToolTypeFunction ToolType = "function"
)

// FunctionDef defines a function tool.
type FunctionDef struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Parameters  interface{} `json:"parameters"` // JSON Schema
}

// Tool represents a tool available to the LLM.
type Tool struct {
	Type     ToolType    `json:"type"`
	Function FunctionDef `json:"function"`
}

// FunctionCall represents a call to a function tool.
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string containing arguments
}

// ToolCall represents a request from the LLM to call a tool.
type ToolCall struct {
	ID       string       `json:"id,omitempty"` // Optional for some providers but good to have
	Type     ToolType     `json:"type"`
	Function FunctionCall `json:"function"`
}

// Message is a single unit of communication.
type Message struct {
	Role       Role       `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"` // Used for tool role messages
}

// ChatRequest encapsulates the input for the LLM.
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Tools       []Tool    `json:"tools,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
}

// ChatResponse encapsulates the output from the LLM.
type ChatResponse struct {
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	Usage     Usage      `json:"usage"`
}

// Usage tracks token consumption.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Provider defines the interface for interacting with LLM backends.
type Provider interface {
	// Chat sends a chat request to the LLM and returns the response.
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
}
