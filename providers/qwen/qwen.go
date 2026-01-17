// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

// Package qwen provides an Alibaba Cloud Qwen API provider for Kairos.
// Qwen uses OpenAI-compatible API format via DashScope.
package qwen

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/jllopis/kairos/pkg/llm"
)

const (
	// DefaultBaseURL is the default DashScope API endpoint.
	DefaultBaseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
)

// Provider implements llm.Provider for Alibaba Cloud Qwen API.
type Provider struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

// Option configures the Provider.
type Option func(*Provider)

// WithModel sets the default model.
func WithModel(model string) Option {
	return func(p *Provider) {
		p.model = model
	}
}

// WithBaseURL sets a custom base URL.
func WithBaseURL(url string) Option {
	return func(p *Provider) {
		p.baseURL = url
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) Option {
	return func(p *Provider) {
		p.client = client
	}
}

// New creates a new Qwen provider.
func New(apiKey string, opts ...Option) *Provider {
	p := &Provider{
		apiKey:  apiKey,
		baseURL: DefaultBaseURL,
		model:   "qwen-plus",
		client:  http.DefaultClient,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Chat implements llm.Provider.
func (p *Provider) Chat(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}

	// Build OpenAI-compatible request
	apiReq := openAIRequest{
		Model:    model,
		Messages: convertMessages(req.Messages),
	}

	if req.Temperature > 0 {
		apiReq.Temperature = &req.Temperature
	}

	if len(req.Tools) > 0 {
		apiReq.Tools = convertTools(req.Tools)
	}

	// Serialize request
	body, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	// Make the request
	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer httpResp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if httpResp.StatusCode != http.StatusOK {
		var errResp errorResponse
		json.Unmarshal(respBody, &errResp)
		return nil, fmt.Errorf("qwen API error (status %d): %s", httpResp.StatusCode, errResp.Error.Message)
	}

	// Parse response
	var apiResp openAIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return convertResponse(&apiResp), nil
}

// OpenAI-compatible request/response types

type openAIRequest struct {
	Model       string           `json:"model"`
	Messages    []openAIMessage  `json:"messages"`
	Tools       []openAITool     `json:"tools,omitempty"`
	Temperature *float64         `json:"temperature,omitempty"`
}

type openAIMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content,omitempty"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type openAITool struct {
	Type     string             `json:"type"`
	Function openAIFunctionDef  `json:"function"`
}

type openAIFunctionDef struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Parameters  interface{} `json:"parameters,omitempty"`
}

type openAIToolCall struct {
	ID       string               `json:"id"`
	Type     string               `json:"type"`
	Function openAIFunctionCall   `json:"function"`
}

type openAIFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openAIResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role      string           `json:"role"`
			Content   string           `json:"content"`
			ToolCalls []openAIToolCall `json:"tool_calls,omitempty"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type errorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// Conversion helpers

func convertMessages(messages []llm.Message) []openAIMessage {
	result := make([]openAIMessage, 0, len(messages))
	for _, msg := range messages {
		oaiMsg := openAIMessage{
			Role:       string(msg.Role),
			Content:    msg.Content,
			ToolCallID: msg.ToolCallID,
		}
		if len(msg.ToolCalls) > 0 {
			oaiMsg.ToolCalls = make([]openAIToolCall, 0, len(msg.ToolCalls))
			for _, tc := range msg.ToolCalls {
				oaiMsg.ToolCalls = append(oaiMsg.ToolCalls, openAIToolCall{
					ID:   tc.ID,
					Type: string(tc.Type),
					Function: openAIFunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				})
			}
		}
		result = append(result, oaiMsg)
	}
	return result
}

func convertTools(tools []llm.Tool) []openAITool {
	result := make([]openAITool, 0, len(tools))
	for _, tool := range tools {
		result = append(result, openAITool{
			Type: string(tool.Type),
			Function: openAIFunctionDef{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				Parameters:  tool.Function.Parameters,
			},
		})
	}
	return result
}

func convertResponse(resp *openAIResponse) *llm.ChatResponse {
	result := &llm.ChatResponse{
		Usage: llm.Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}

	if len(resp.Choices) > 0 {
		choice := resp.Choices[0]
		result.Content = choice.Message.Content

		if len(choice.Message.ToolCalls) > 0 {
			result.ToolCalls = make([]llm.ToolCall, 0, len(choice.Message.ToolCalls))
			for _, tc := range choice.Message.ToolCalls {
				result.ToolCalls = append(result.ToolCalls, llm.ToolCall{
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

	return result
}

// Ensure Provider implements llm.Provider.
var _ llm.Provider = (*Provider)(nil)
