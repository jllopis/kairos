// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

// Package anthropic provides an Anthropic Claude API provider for Kairos.
package anthropic

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/jllopis/kairos/pkg/llm"
)

// Provider implements llm.Provider for Anthropic Claude API.
type Provider struct {
	client    anthropic.Client
	model     string
	maxTokens int64
}

// Option configures the Provider.
type Option func(*Provider)

// WithModel sets the default model.
func WithModel(model string) Option {
	return func(p *Provider) {
		p.model = model
	}
}

// WithMaxTokens sets the maximum tokens for responses.
func WithMaxTokens(tokens int64) Option {
	return func(p *Provider) {
		p.maxTokens = tokens
	}
}

// WithBaseURL sets a custom base URL.
func WithBaseURL(url string) Option {
	return func(p *Provider) {
		p.client = anthropic.NewClient(option.WithBaseURL(url))
	}
}

// WithAPIKey sets the API key.
func WithAPIKey(apiKey string) Option {
	return func(p *Provider) {
		p.client = anthropic.NewClient(option.WithAPIKey(apiKey))
	}
}

// New creates a new Anthropic provider.
// API key is read from ANTHROPIC_API_KEY environment variable by default.
func New(opts ...Option) *Provider {
	p := &Provider{
		client:    anthropic.NewClient(),
		model:     "claude-haiku-4-20250514",
		maxTokens: 4096,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// NewWithAPIKey creates a new Anthropic provider with explicit API key.
func NewWithAPIKey(apiKey string, opts ...Option) *Provider {
	opts = append([]Option{WithAPIKey(apiKey)}, opts...)
	return New(opts...)
}

// Chat implements llm.Provider.
func (p *Provider) Chat(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}

	// Extract system message and convert other messages
	var systemPrompt string
	messages := make([]anthropic.MessageParam, 0, len(req.Messages))

	for _, msg := range req.Messages {
		if msg.Role == llm.RoleSystem {
			systemPrompt = msg.Content
			continue
		}
		messages = append(messages, convertMessage(msg))
	}

	// Build request params
	params := anthropic.MessageNewParams{
		Model:     model,
		MaxTokens: p.maxTokens,
		Messages:  messages,
	}

	// Add system prompt if present
	if systemPrompt != "" {
		params.System = []anthropic.TextBlockParam{
			{Type: "text", Text: systemPrompt},
		}
	}

	// Add temperature if set
	if req.Temperature > 0 {
		params.Temperature = anthropic.Float(req.Temperature)
	}

	// Add tools if present
	if len(req.Tools) > 0 {
		tools := make([]anthropic.ToolUnionParam, 0, len(req.Tools))
		for _, tool := range req.Tools {
			tools = append(tools, convertTool(tool))
		}
		params.Tools = tools
	}

	// Make the API call
	message, err := p.client.Messages.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("anthropic message failed: %w", err)
	}

	// Convert response
	return convertResponse(message), nil
}

// convertMessage converts Kairos message to Anthropic format.
func convertMessage(msg llm.Message) anthropic.MessageParam {
	switch msg.Role {
	case llm.RoleUser:
		return anthropic.NewUserMessage(anthropic.NewTextBlock(msg.Content))
	case llm.RoleAssistant:
		if len(msg.ToolCalls) > 0 {
			// Assistant message with tool use
			blocks := make([]anthropic.ContentBlockParamUnion, 0, len(msg.ToolCalls)+1)
			if msg.Content != "" {
				blocks = append(blocks, anthropic.NewTextBlock(msg.Content))
			}
			for _, tc := range msg.ToolCalls {
				var input map[string]interface{}
				json.Unmarshal([]byte(tc.Function.Arguments), &input)
				blocks = append(blocks, anthropic.NewToolUseBlock(tc.ID, input, tc.Function.Name))
			}
			return anthropic.MessageParam{
				Role:    "assistant",
				Content: blocks,
			}
		}
		return anthropic.NewAssistantMessage(anthropic.NewTextBlock(msg.Content))
	case llm.RoleTool:
		// Tool result message - Anthropic requires tool results as user messages
		return anthropic.NewUserMessage(
			anthropic.NewToolResultBlock(msg.ToolCallID, msg.Content, false),
		)
	default:
		return anthropic.NewUserMessage(anthropic.NewTextBlock(msg.Content))
	}
}

// convertTool converts Kairos tool to Anthropic format.
func convertTool(tool llm.Tool) anthropic.ToolUnionParam {
	// Convert parameters to Anthropic input schema
	paramsJSON, _ := json.Marshal(tool.Function.Parameters)
	var inputSchema anthropic.ToolInputSchemaParam
	json.Unmarshal(paramsJSON, &inputSchema)

	return anthropic.ToolUnionParam{
		OfTool: &anthropic.ToolParam{
			Name:        tool.Function.Name,
			Description: anthropic.String(tool.Function.Description),
			InputSchema: inputSchema,
		},
	}
}

// convertResponse converts Anthropic response to Kairos format.
func convertResponse(message *anthropic.Message) *llm.ChatResponse {
	resp := &llm.ChatResponse{
		Usage: llm.Usage{
			PromptTokens:     int(message.Usage.InputTokens),
			CompletionTokens: int(message.Usage.OutputTokens),
			TotalTokens:      int(message.Usage.InputTokens + message.Usage.OutputTokens),
		},
	}

	// Process content blocks
	var textContent string
	var toolCalls []llm.ToolCall

	for _, block := range message.Content {
		switch block.Type {
		case "text":
			textContent += block.Text
		case "tool_use":
			// Convert input back to JSON string
			argsJSON, _ := json.Marshal(block.Input)
			toolCalls = append(toolCalls, llm.ToolCall{
				ID:   block.ID,
				Type: llm.ToolTypeFunction,
				Function: llm.FunctionCall{
					Name:      block.Name,
					Arguments: string(argsJSON),
				},
			})
		}
	}

	resp.Content = textContent
	resp.ToolCalls = toolCalls

	return resp
}

// ChatStream implements llm.StreamingProvider for streaming responses.
func (p *Provider) ChatStream(ctx context.Context, req llm.ChatRequest) (<-chan llm.StreamChunk, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}

	// Extract system message and convert other messages
	var systemPrompt string
	messages := make([]anthropic.MessageParam, 0, len(req.Messages))

	for _, msg := range req.Messages {
		if msg.Role == llm.RoleSystem {
			systemPrompt = msg.Content
			continue
		}
		messages = append(messages, convertMessage(msg))
	}

	// Build request params
	params := anthropic.MessageNewParams{
		Model:     model,
		MaxTokens: p.maxTokens,
		Messages:  messages,
	}

	// Add system prompt if present
	if systemPrompt != "" {
		params.System = []anthropic.TextBlockParam{
			{Type: "text", Text: systemPrompt},
		}
	}

	// Add temperature if set
	if req.Temperature > 0 {
		params.Temperature = anthropic.Float(req.Temperature)
	}

	// Add tools if present
	if len(req.Tools) > 0 {
		tools := make([]anthropic.ToolUnionParam, 0, len(req.Tools))
		for _, tool := range req.Tools {
			tools = append(tools, convertTool(tool))
		}
		params.Tools = tools
	}

	// Create streaming request
	stream := p.client.Messages.NewStreaming(ctx, params)

	// Create output channel
	chunks := make(chan llm.StreamChunk, 100)

	// Process stream in goroutine
	go func() {
		defer close(chunks)

		var toolCalls []llm.ToolCall
		currentToolCall := (*llm.ToolCall)(nil)
		var toolInputJSON string

		for stream.Next() {
			event := stream.Current()

			chunk := llm.StreamChunk{}

			switch event.Type {
			case "content_block_delta":
				if event.Delta.Type == "text_delta" {
					chunk.Content = event.Delta.Text
				} else if event.Delta.Type == "input_json_delta" {
					toolInputJSON += event.Delta.PartialJSON
				}

			case "content_block_start":
				if event.ContentBlock.Type == "tool_use" {
					currentToolCall = &llm.ToolCall{
						ID:   event.ContentBlock.ID,
						Type: llm.ToolTypeFunction,
						Function: llm.FunctionCall{
							Name: event.ContentBlock.Name,
						},
					}
					toolInputJSON = ""
				}

			case "content_block_stop":
				if currentToolCall != nil {
					currentToolCall.Function.Arguments = toolInputJSON
					toolCalls = append(toolCalls, *currentToolCall)
					currentToolCall = nil
					toolInputJSON = ""
				}

			case "message_stop":
				chunk.Done = true
				chunk.ToolCalls = toolCalls

			case "message_delta":
				if event.Usage.OutputTokens > 0 {
					chunk.Usage = &llm.Usage{
						CompletionTokens: int(event.Usage.OutputTokens),
					}
				}
			}

			select {
			case chunks <- chunk:
			case <-ctx.Done():
				chunks <- llm.StreamChunk{Error: ctx.Err()}
				return
			}
		}

		if err := stream.Err(); err != nil {
			chunks <- llm.StreamChunk{Error: err}
		}
	}()

	return chunks, nil
}

// Ensure Provider implements llm.Provider.
var _ llm.Provider = (*Provider)(nil)

// Ensure Provider implements llm.StreamingProvider.
var _ llm.StreamingProvider = (*Provider)(nil)
