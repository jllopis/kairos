// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

// Package openai provides an OpenAI API provider for Kairos.
package openai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jllopis/kairos/pkg/llm"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/packages/param"
)

// Provider implements llm.Provider for OpenAI API.
type Provider struct {
	client openai.Client
	model  string
}

// Option configures the Provider.
type Option func(*Provider)

// WithModel sets the default model.
func WithModel(model string) Option {
	return func(p *Provider) {
		p.model = model
	}
}

// WithBaseURL sets a custom base URL (for Azure OpenAI or proxies).
func WithBaseURL(url string) Option {
	return func(p *Provider) {
		p.client = openai.NewClient(option.WithBaseURL(url))
	}
}

// WithAPIKey sets the API key.
func WithAPIKey(apiKey string) Option {
	return func(p *Provider) {
		p.client = openai.NewClient(option.WithAPIKey(apiKey))
	}
}

// New creates a new OpenAI provider.
// API key is read from OPENAI_API_KEY environment variable by default.
func New(opts ...Option) *Provider {
	p := &Provider{
		client: openai.NewClient(),
		model:  "gpt-5-mini",
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// NewWithAPIKey creates a new OpenAI provider with explicit API key.
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

	// Convert messages
	messages := make([]openai.ChatCompletionMessageParamUnion, 0, len(req.Messages))
	for _, msg := range req.Messages {
		messages = append(messages, convertMessage(msg))
	}

	// Build request params
	params := openai.ChatCompletionNewParams{
		Model:    model,
		Messages: messages,
	}

	// Add temperature if set
	if req.Temperature > 0 {
		params.Temperature = openai.Float(req.Temperature)
	}

	// Add tools if present
	if len(req.Tools) > 0 {
		tools := make([]openai.ChatCompletionToolParam, 0, len(req.Tools))
		for _, tool := range req.Tools {
			tools = append(tools, convertTool(tool))
		}
		params.Tools = tools
	}

	// Make the API call
	completion, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("openai chat completion failed: %w", err)
	}

	// Convert response
	return convertResponse(completion), nil
}

// convertMessage converts Kairos message to OpenAI format.
func convertMessage(msg llm.Message) openai.ChatCompletionMessageParamUnion {
	switch msg.Role {
	case llm.RoleSystem:
		return openai.SystemMessage(msg.Content)
	case llm.RoleUser:
		return openai.UserMessage(msg.Content)
	case llm.RoleAssistant:
		if len(msg.ToolCalls) > 0 {
			toolCalls := make([]openai.ChatCompletionMessageToolCallParam, 0, len(msg.ToolCalls))
			for _, tc := range msg.ToolCalls {
				toolCalls = append(toolCalls, openai.ChatCompletionMessageToolCallParam{
					ID:   tc.ID,
					Type: "function",
					Function: openai.ChatCompletionMessageToolCallFunctionParam{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				})
			}
			assistantMsg := openai.ChatCompletionAssistantMessageParam{
				ToolCalls: toolCalls,
			}
			if msg.Content != "" {
				assistantMsg.Content = openai.ChatCompletionAssistantMessageParamContentUnion{
					OfString: param.NewOpt(msg.Content),
				}
			}
			return openai.ChatCompletionMessageParamUnion{
				OfAssistant: &assistantMsg,
			}
		}
		return openai.AssistantMessage(msg.Content)
	case llm.RoleTool:
		return openai.ToolMessage(msg.Content, msg.ToolCallID)
	default:
		return openai.UserMessage(msg.Content)
	}
}

// convertTool converts Kairos tool to OpenAI format.
func convertTool(tool llm.Tool) openai.ChatCompletionToolParam {
	// Convert parameters to raw JSON
	paramsJSON, _ := json.Marshal(tool.Function.Parameters)
	var params openai.FunctionParameters
	json.Unmarshal(paramsJSON, &params)

	return openai.ChatCompletionToolParam{
		Type: "function",
		Function: openai.FunctionDefinitionParam{
			Name:        tool.Function.Name,
			Description: openai.String(tool.Function.Description),
			Parameters:  params,
		},
	}
}

// convertResponse converts OpenAI response to Kairos format.
func convertResponse(completion *openai.ChatCompletion) *llm.ChatResponse {
	resp := &llm.ChatResponse{
		Usage: llm.Usage{
			PromptTokens:     int(completion.Usage.PromptTokens),
			CompletionTokens: int(completion.Usage.CompletionTokens),
			TotalTokens:      int(completion.Usage.TotalTokens),
		},
	}

	if len(completion.Choices) > 0 {
		choice := completion.Choices[0]
		resp.Content = choice.Message.Content

		// Convert tool calls
		if len(choice.Message.ToolCalls) > 0 {
			resp.ToolCalls = make([]llm.ToolCall, 0, len(choice.Message.ToolCalls))
			for _, tc := range choice.Message.ToolCalls {
				resp.ToolCalls = append(resp.ToolCalls, llm.ToolCall{
					ID:   tc.ID,
					Type: llm.ToolTypeFunction,
					Function: llm.FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				})
			}
		}
	}

	return resp
}

// ChatStream implements llm.StreamingProvider for streaming responses.
func (p *Provider) ChatStream(ctx context.Context, req llm.ChatRequest) (<-chan llm.StreamChunk, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}

	// Convert messages
	messages := make([]openai.ChatCompletionMessageParamUnion, 0, len(req.Messages))
	for _, msg := range req.Messages {
		messages = append(messages, convertMessage(msg))
	}

	// Build request params
	params := openai.ChatCompletionNewParams{
		Model:    model,
		Messages: messages,
	}

	// Add temperature if set
	if req.Temperature > 0 {
		params.Temperature = openai.Float(req.Temperature)
	}

	// Add tools if present
	if len(req.Tools) > 0 {
		tools := make([]openai.ChatCompletionToolParam, 0, len(req.Tools))
		for _, tool := range req.Tools {
			tools = append(tools, convertTool(tool))
		}
		params.Tools = tools
	}

	// Create streaming request
	stream := p.client.Chat.Completions.NewStreaming(ctx, params)

	// Create output channel
	chunks := make(chan llm.StreamChunk, 100)

	// Process stream in goroutine
	go func() {
		defer close(chunks)

		var accumulatedToolCalls []llm.ToolCall
		toolCallsMap := make(map[int]*llm.ToolCall) // Track tool calls by index

		for stream.Next() {
			event := stream.Current()

			chunk := llm.StreamChunk{}

			if len(event.Choices) > 0 {
				delta := event.Choices[0].Delta

				// Content delta
				if delta.Content != "" {
					chunk.Content = delta.Content
				}

				// Tool calls delta
				for _, tc := range delta.ToolCalls {
					idx := int(tc.Index)
					if _, exists := toolCallsMap[idx]; !exists {
						toolCallsMap[idx] = &llm.ToolCall{
							ID:   tc.ID,
							Type: llm.ToolTypeFunction,
							Function: llm.FunctionCall{
								Name: tc.Function.Name,
							},
						}
					}
					// Accumulate arguments
					toolCallsMap[idx].Function.Arguments += tc.Function.Arguments
				}

				// Check if done
				if event.Choices[0].FinishReason != "" {
					chunk.Done = true
					// Convert map to slice
					for i := 0; i < len(toolCallsMap); i++ {
						if tc, ok := toolCallsMap[i]; ok {
							accumulatedToolCalls = append(accumulatedToolCalls, *tc)
						}
					}
					chunk.ToolCalls = accumulatedToolCalls
				}
			}

			// Usage info (if available in stream)
			if event.Usage.TotalTokens > 0 {
				chunk.Usage = &llm.Usage{
					PromptTokens:     int(event.Usage.PromptTokens),
					CompletionTokens: int(event.Usage.CompletionTokens),
					TotalTokens:      int(event.Usage.TotalTokens),
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
