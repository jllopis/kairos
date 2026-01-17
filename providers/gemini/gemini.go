// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

// Package gemini provides a Google Gemini API provider for Kairos.
package gemini

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jllopis/kairos/pkg/llm"
	"google.golang.org/genai"
)

// Provider implements llm.Provider for Google Gemini API.
type Provider struct {
	client *genai.Client
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

// New creates a new Gemini provider.
// API key is read from GOOGLE_API_KEY or GEMINI_API_KEY environment variable by default.
func New(ctx context.Context, opts ...Option) (*Provider, error) {
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	p := &Provider{
		client: client,
		model:  "gemini-3-flash-preview",
	}
	for _, opt := range opts {
		opt(p)
	}
	return p, nil
}

// NewWithAPIKey creates a new Gemini provider with explicit API key.
func NewWithAPIKey(ctx context.Context, apiKey string, opts ...Option) (*Provider, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	p := &Provider{
		client: client,
		model:  "gemini-3-flash-preview",
	}
	for _, opt := range opts {
		opt(p)
	}
	return p, nil
}

// Chat implements llm.Provider.
func (p *Provider) Chat(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}

	// Build contents from messages
	contents, systemInstruction := convertMessages(req.Messages)

	// Build config
	config := &genai.GenerateContentConfig{}

	if systemInstruction != "" {
		config.SystemInstruction = &genai.Content{
			Parts: []*genai.Part{{Text: systemInstruction}},
		}
	}

	if req.Temperature > 0 {
		temp := float32(req.Temperature)
		config.Temperature = &temp
	}

	// Add tools if present
	if len(req.Tools) > 0 {
		config.Tools = []*genai.Tool{
			{FunctionDeclarations: convertTools(req.Tools)},
		}
	}

	// Make the API call
	resp, err := p.client.Models.GenerateContent(ctx, model, contents, config)
	if err != nil {
		return nil, fmt.Errorf("gemini generate content failed: %w", err)
	}

	return convertResponse(resp), nil
}

// Close is a no-op as the Gemini client doesn't require explicit closing.
func (p *Provider) Close() error {
	return nil
}

// convertMessages converts Kairos messages to Gemini format.
func convertMessages(messages []llm.Message) ([]*genai.Content, string) {
	var systemInstruction string
	contents := make([]*genai.Content, 0, len(messages))

	for _, msg := range messages {
		switch msg.Role {
		case llm.RoleSystem:
			systemInstruction = msg.Content
		case llm.RoleUser:
			contents = append(contents, &genai.Content{
				Role:  "user",
				Parts: []*genai.Part{{Text: msg.Content}},
			})
		case llm.RoleAssistant:
			content := &genai.Content{
				Role:  "model",
				Parts: []*genai.Part{},
			}
			if msg.Content != "" {
				content.Parts = append(content.Parts, &genai.Part{Text: msg.Content})
			}
			// Add function calls
			for _, tc := range msg.ToolCalls {
				var args map[string]interface{}
				json.Unmarshal([]byte(tc.Function.Arguments), &args)
				content.Parts = append(content.Parts, &genai.Part{
					FunctionCall: &genai.FunctionCall{
						Name: tc.Function.Name,
						Args: args,
					},
				})
			}
			contents = append(contents, content)
		case llm.RoleTool:
			// Function response
			var result map[string]interface{}
			// Try to parse as JSON, otherwise wrap in response object
			if err := json.Unmarshal([]byte(msg.Content), &result); err != nil {
				result = map[string]interface{}{"result": msg.Content}
			}
			contents = append(contents, &genai.Content{
				Role: "user",
				Parts: []*genai.Part{
					{
						FunctionResponse: &genai.FunctionResponse{
							Name:     msg.ToolCallID, // Gemini uses name, we store it in ToolCallID
							Response: result,
						},
					},
				},
			})
		}
	}

	return contents, systemInstruction
}

// convertTools converts Kairos tools to Gemini function declarations.
func convertTools(tools []llm.Tool) []*genai.FunctionDeclaration {
	declarations := make([]*genai.FunctionDeclaration, 0, len(tools))

	for _, tool := range tools {
		// Convert parameters to Gemini schema
		paramsJSON, _ := json.Marshal(tool.Function.Parameters)
		var schema *genai.Schema
		json.Unmarshal(paramsJSON, &schema)

		declarations = append(declarations, &genai.FunctionDeclaration{
			Name:        tool.Function.Name,
			Description: tool.Function.Description,
			Parameters:  schema,
		})
	}

	return declarations
}

// convertResponse converts Gemini response to Kairos format.
func convertResponse(resp *genai.GenerateContentResponse) *llm.ChatResponse {
	result := &llm.ChatResponse{}

	// Get usage metadata
	if resp.UsageMetadata != nil {
		result.Usage = llm.Usage{
			PromptTokens:     int(resp.UsageMetadata.PromptTokenCount),
			CompletionTokens: int(resp.UsageMetadata.CandidatesTokenCount),
			TotalTokens:      int(resp.UsageMetadata.TotalTokenCount),
		}
	}

	// Process candidates
	if len(resp.Candidates) > 0 {
		candidate := resp.Candidates[0]
		if candidate.Content != nil {
			for _, part := range candidate.Content.Parts {
				if part.Text != "" {
					result.Content += part.Text
				}
				if part.FunctionCall != nil {
					argsJSON, _ := json.Marshal(part.FunctionCall.Args)
					result.ToolCalls = append(result.ToolCalls, llm.ToolCall{
						ID:   part.FunctionCall.Name, // Gemini doesn't have separate IDs
						Type: llm.ToolTypeFunction,
						Function: llm.FunctionCall{
							Name:      part.FunctionCall.Name,
							Arguments: string(argsJSON),
						},
					})
				}
			}
		}
	}

	return result
}

// ChatStream implements llm.StreamingProvider for streaming responses.
func (p *Provider) ChatStream(ctx context.Context, req llm.ChatRequest) (<-chan llm.StreamChunk, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}

	// Build contents from messages
	contents, systemInstruction := convertMessages(req.Messages)

	// Build config
	config := &genai.GenerateContentConfig{}

	if systemInstruction != "" {
		config.SystemInstruction = &genai.Content{
			Parts: []*genai.Part{{Text: systemInstruction}},
		}
	}

	if req.Temperature > 0 {
		temp := float32(req.Temperature)
		config.Temperature = &temp
	}

	// Add tools if present
	if len(req.Tools) > 0 {
		config.Tools = []*genai.Tool{
			{FunctionDeclarations: convertTools(req.Tools)},
		}
	}

	// Create output channel
	chunks := make(chan llm.StreamChunk, 100)

	// Process stream in goroutine
	go func() {
		defer close(chunks)

		iter := p.client.Models.GenerateContentStream(ctx, model, contents, config)

		// iter.Seq2 is a function that takes a yield callback
		iter(func(resp *genai.GenerateContentResponse, err error) bool {
			if err != nil {
				chunks <- llm.StreamChunk{Error: err}
				return false // stop iteration
			}

			chunk := llm.StreamChunk{}

			// Get usage metadata
			if resp.UsageMetadata != nil {
				chunk.Usage = &llm.Usage{
					PromptTokens:     int(resp.UsageMetadata.PromptTokenCount),
					CompletionTokens: int(resp.UsageMetadata.CandidatesTokenCount),
					TotalTokens:      int(resp.UsageMetadata.TotalTokenCount),
				}
			}

			// Process candidates
			if len(resp.Candidates) > 0 {
				candidate := resp.Candidates[0]
				if candidate.Content != nil {
					for _, part := range candidate.Content.Parts {
						if part.Text != "" {
							chunk.Content += part.Text
						}
						if part.FunctionCall != nil {
							argsJSON, _ := json.Marshal(part.FunctionCall.Args)
							chunk.ToolCalls = append(chunk.ToolCalls, llm.ToolCall{
								ID:   part.FunctionCall.Name,
								Type: llm.ToolTypeFunction,
								Function: llm.FunctionCall{
									Name:      part.FunctionCall.Name,
									Arguments: string(argsJSON),
								},
							})
						}
					}
				}

				// Check finish reason
				if candidate.FinishReason != "" {
					chunk.Done = true
				}
			}

			select {
			case chunks <- chunk:
				return true // continue iteration
			case <-ctx.Done():
				chunks <- llm.StreamChunk{Error: ctx.Err()}
				return false // stop iteration
			}
		})

		// Send final done chunk if not already sent
		select {
		case chunks <- llm.StreamChunk{Done: true}:
		default:
		}
	}()

	return chunks, nil
}

// Ensure Provider implements llm.Provider.
var _ llm.Provider = (*Provider)(nil)

// Ensure Provider implements llm.StreamingProvider.
var _ llm.StreamingProvider = (*Provider)(nil)
