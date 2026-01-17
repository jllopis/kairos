// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package testing

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/jllopis/kairos/pkg/llm"
)

// ScenarioProvider is an enhanced mock provider for testing scenarios.
// It supports scripted responses, tool call simulation, and request capture.
type ScenarioProvider struct {
	mu            sync.Mutex
	responses     []ScriptedResponse
	currentIndex  int
	requests      []llm.ChatRequest
	defaultError  error
	onChat        func(req llm.ChatRequest) (*llm.ChatResponse, error)
}

// ScriptedResponse defines a response for the scenario provider.
type ScriptedResponse struct {
	Content   string
	ToolCalls []llm.ToolCall
	Error     error
	Usage     llm.Usage
	// Condition allows conditional responses based on request
	Condition func(req llm.ChatRequest) bool
}

// NewScenarioProvider creates a new scenario provider.
func NewScenarioProvider() *ScenarioProvider {
	return &ScenarioProvider{
		responses: make([]ScriptedResponse, 0),
		requests:  make([]llm.ChatRequest, 0),
	}
}

// AddResponse queues a response to be returned.
func (p *ScenarioProvider) AddResponse(content string) *ScenarioProvider {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.responses = append(p.responses, ScriptedResponse{Content: content})
	return p
}

// AddToolCallResponse queues a response with tool calls.
func (p *ScenarioProvider) AddToolCallResponse(toolCalls ...llm.ToolCall) *ScenarioProvider {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.responses = append(p.responses, ScriptedResponse{ToolCalls: toolCalls})
	return p
}

// AddErrorResponse queues an error response.
func (p *ScenarioProvider) AddErrorResponse(err error) *ScenarioProvider {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.responses = append(p.responses, ScriptedResponse{Error: err})
	return p
}

// AddScriptedResponse adds a fully configured response.
func (p *ScenarioProvider) AddScriptedResponse(resp ScriptedResponse) *ScenarioProvider {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.responses = append(p.responses, resp)
	return p
}

// WithDefaultError sets the error to return when no responses are queued.
func (p *ScenarioProvider) WithDefaultError(err error) *ScenarioProvider {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.defaultError = err
	return p
}

// WithChatFunc sets a custom function for handling chat requests.
func (p *ScenarioProvider) WithChatFunc(fn func(req llm.ChatRequest) (*llm.ChatResponse, error)) *ScenarioProvider {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onChat = fn
	return p
}

// Chat implements llm.Provider.
func (p *ScenarioProvider) Chat(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Record request
	p.requests = append(p.requests, req)

	// Check for custom handler
	if p.onChat != nil {
		return p.onChat(req)
	}

	// Check for queued responses
	if p.currentIndex >= len(p.responses) {
		if p.defaultError != nil {
			return nil, p.defaultError
		}
		return nil, fmt.Errorf("no more scripted responses (call %d)", p.currentIndex+1)
	}

	resp := p.responses[p.currentIndex]
	p.currentIndex++

	// Check condition if present
	if resp.Condition != nil && !resp.Condition(req) {
		// Skip to next response that matches
		for p.currentIndex < len(p.responses) {
			resp = p.responses[p.currentIndex]
			p.currentIndex++
			if resp.Condition == nil || resp.Condition(req) {
				break
			}
		}
	}

	if resp.Error != nil {
		return nil, resp.Error
	}

	return &llm.ChatResponse{
		Content:   resp.Content,
		ToolCalls: resp.ToolCalls,
		Usage:     resp.Usage,
	}, nil
}

// Requests returns all captured requests.
func (p *ScenarioProvider) Requests() []llm.ChatRequest {
	p.mu.Lock()
	defer p.mu.Unlock()
	result := make([]llm.ChatRequest, len(p.requests))
	copy(result, p.requests)
	return result
}

// LastRequest returns the most recent request.
func (p *ScenarioProvider) LastRequest() *llm.ChatRequest {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.requests) == 0 {
		return nil
	}
	req := p.requests[len(p.requests)-1]
	return &req
}

// CallCount returns the number of Chat calls made.
func (p *ScenarioProvider) CallCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.requests)
}

// Reset clears all state.
func (p *ScenarioProvider) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.currentIndex = 0
	p.requests = p.requests[:0]
}

// ToolCallBuilder helps construct tool calls for testing.
type ToolCallBuilder struct {
	id       string
	name     string
	args     map[string]any
}

// NewToolCall creates a new tool call builder.
func NewToolCall(name string) *ToolCallBuilder {
	return &ToolCallBuilder{
		name: name,
		args: make(map[string]any),
	}
}

// WithID sets the tool call ID.
func (b *ToolCallBuilder) WithID(id string) *ToolCallBuilder {
	b.id = id
	return b
}

// WithArg adds an argument to the tool call.
func (b *ToolCallBuilder) WithArg(key string, value any) *ToolCallBuilder {
	b.args[key] = value
	return b
}

// WithArgs sets all arguments at once.
func (b *ToolCallBuilder) WithArgs(args map[string]any) *ToolCallBuilder {
	b.args = args
	return b
}

// Build creates the tool call.
func (b *ToolCallBuilder) Build() llm.ToolCall {
	argsJSON, _ := json.Marshal(b.args)
	return llm.ToolCall{
		ID:   b.id,
		Type: llm.ToolTypeFunction,
		Function: llm.FunctionCall{
			Name:      b.name,
			Arguments: string(argsJSON),
		},
	}
}

// ToolDefinitionBuilder helps construct tool definitions for testing.
type ToolDefinitionBuilder struct {
	name        string
	description string
	parameters  map[string]any
}

// NewToolDefinition creates a new tool definition builder.
func NewToolDefinition(name string) *ToolDefinitionBuilder {
	return &ToolDefinitionBuilder{
		name: name,
		parameters: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}
}

// WithDescription sets the tool description.
func (b *ToolDefinitionBuilder) WithDescription(desc string) *ToolDefinitionBuilder {
	b.description = desc
	return b
}

// WithParameter adds a parameter to the tool definition.
func (b *ToolDefinitionBuilder) WithParameter(name, paramType, description string, required bool) *ToolDefinitionBuilder {
	props := b.parameters["properties"].(map[string]any)
	props[name] = map[string]any{
		"type":        paramType,
		"description": description,
	}
	if required {
		reqs, ok := b.parameters["required"].([]string)
		if !ok {
			reqs = []string{}
		}
		b.parameters["required"] = append(reqs, name)
	}
	return b
}

// Build creates the tool definition.
func (b *ToolDefinitionBuilder) Build() llm.Tool {
	return llm.Tool{
		Type: llm.ToolTypeFunction,
		Function: llm.FunctionDef{
			Name:        b.name,
			Description: b.description,
			Parameters:  b.parameters,
		},
	}
}
