// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package scaffold

// Archetype-specific templates

const toolsTemplate = `// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

// Package tools defines the tools available to the agent.
package tools

import (
	"context"
	"fmt"

	"github.com/jllopis/kairos/pkg/core"
)

// GetTools returns all tools for this agent.
func GetTools() []core.Tool {
	return []core.Tool{
		&EchoTool{},
		// Add your tools here
	}
}

// EchoTool is a simple example tool.
type EchoTool struct{}

func (t *EchoTool) Name() string {
	return "echo"
}

func (t *EchoTool) Description() string {
	return "Echoes back the input message"
}

func (t *EchoTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"message": map[string]any{
				"type":        "string",
				"description": "The message to echo",
			},
		},
		"required": []string{"message"},
	}
}

func (t *EchoTool) Call(ctx context.Context, args map[string]any) (any, error) {
	message, ok := args["message"].(string)
	if !ok {
		return nil, fmt.Errorf("message must be a string")
	}
	return fmt.Sprintf("Echo: %s", message), nil
}
`

const plannerTemplate = `// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

// Package planner defines the execution graph for the coordinator.
package planner

import (
	"context"

	"github.com/jllopis/kairos/pkg/planner"
)

// GetGraph returns the planner graph for this coordinator.
func GetGraph() *planner.Graph {
	// Define your execution graph here
	// This is a simple example - customize for your use case
	
	nodes := []planner.Node{
		{ID: "start", Type: "init", Next: []string{"process"}},
		{ID: "process", Type: "llm_call", Next: []string{"end"}},
		{ID: "end", Type: "terminal", Next: nil},
	}
	
	graph, _ := planner.NewGraph(nodes)
	return graph
}

// GetHandlers returns the handlers for each node type.
func GetHandlers() map[string]planner.Handler {
	return map[string]planner.Handler{
		"init": func(ctx context.Context, node planner.Node, state *planner.State) (any, error) {
			return "initialized", nil
		},
		"llm_call": func(ctx context.Context, node planner.Node, state *planner.State) (any, error) {
			// TODO: Call LLM here
			return "processed", nil
		},
		"terminal": func(ctx context.Context, node planner.Node, state *planner.State) (any, error) {
			return state.Get("process"), nil
		},
	}
}
`

const policiesTemplate = `// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

// Package policies defines governance rules for the agent.
package policies

import (
	"github.com/jllopis/kairos/pkg/governance"
)

// GetPolicyEngine returns a strict policy engine for this agent.
func GetPolicyEngine() governance.PolicyEngine {
	// Default deny - only explicitly allowed patterns pass
	rules := []governance.Rule{
		// Read operations allowed
		{ID: "allow-read", Effect: "allow", Name: "read_*"},
		{ID: "allow-get", Effect: "allow", Name: "get_*"},
		{ID: "allow-search", Effect: "allow", Name: "search_*"},
		{ID: "allow-list", Effect: "allow", Name: "list_*"},
		
		// Write operations denied
		{ID: "deny-write", Effect: "deny", Name: "write_*"},
		{ID: "deny-delete", Effect: "deny", Name: "delete_*"},
		{ID: "deny-update", Effect: "deny", Name: "update_*"},
		
		// Admin operations blocked
		{ID: "deny-admin", Effect: "deny", Name: "admin_*"},
	}
	
	return governance.NewRuleSet(rules)
}

// ExplainPolicy returns a human-readable explanation of the policies.
func ExplainPolicy() string {
	return ` + "`" + `
Policy: Strict Read-Only

Allowed:
  - read_*, get_*, search_*, list_* operations

Denied:
  - write_*, delete_*, update_* operations
  - admin_* operations
` + "`" + `
}
`
