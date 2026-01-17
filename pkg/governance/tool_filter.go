// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package governance

import (
	"context"
	"path"
	"strings"
)

// ToolFilter provides tool-level filtering based on allowlists and policies.
// This centralizes tool access control that was previously scattered in agent code.
type ToolFilter struct {
	allowlist    map[string]bool
	denylist     map[string]bool
	policyEngine PolicyEngine
}

// ToolFilterOption configures a ToolFilter.
type ToolFilterOption func(*ToolFilter)

// NewToolFilter creates a new ToolFilter with the given options.
func NewToolFilter(opts ...ToolFilterOption) *ToolFilter {
	tf := &ToolFilter{
		allowlist: make(map[string]bool),
		denylist:  make(map[string]bool),
	}
	for _, opt := range opts {
		opt(tf)
	}
	return tf
}

// WithAllowlist sets the allowlist of permitted tool names/patterns.
func WithAllowlist(tools []string) ToolFilterOption {
	return func(tf *ToolFilter) {
		for _, tool := range tools {
			tool = strings.TrimSpace(tool)
			if tool != "" {
				tf.allowlist[tool] = true
			}
		}
	}
}

// WithDenylist sets the denylist of forbidden tool names/patterns.
func WithDenylist(tools []string) ToolFilterOption {
	return func(tf *ToolFilter) {
		for _, tool := range tools {
			tool = strings.TrimSpace(tool)
			if tool != "" {
				tf.denylist[tool] = true
			}
		}
	}
}

// WithPolicyEngine attaches a policy engine for additional evaluation.
func WithPolicyEngine(engine PolicyEngine) ToolFilterOption {
	return func(tf *ToolFilter) {
		tf.policyEngine = engine
	}
}

// IsAllowed checks if a tool name is permitted by the filter.
// Evaluation order:
// 1. If denylist contains tool → deny
// 2. If allowlist is non-empty and doesn't contain tool → deny
// 3. If policy engine exists, evaluate → respect decision
// 4. Otherwise → allow
func (tf *ToolFilter) IsAllowed(ctx context.Context, toolName string) Decision {
	// Check denylist first (explicit denies take precedence)
	if tf.matchesList(toolName, tf.denylist) {
		return Decision{
			Allowed: false,
			Status:  DecisionStatusDeny,
			Reason:  "tool is in denylist",
		}
	}

	// Check allowlist (if non-empty, tool must be in it)
	if len(tf.allowlist) > 0 && !tf.matchesList(toolName, tf.allowlist) {
		return Decision{
			Allowed: false,
			Status:  DecisionStatusDeny,
			Reason:  "tool is not in allowlist",
		}
	}

	// Check policy engine if available
	if tf.policyEngine != nil {
		action := Action{
			Type: ActionTool,
			Name: toolName,
		}
		return tf.policyEngine.Evaluate(ctx, action)
	}

	// Default: allow
	return Decision{
		Allowed: true,
		Status:  DecisionStatusAllow,
	}
}

// FilterTools returns only the tools that pass the filter.
// toolNames is a slice of tool names, returns filtered names.
func (tf *ToolFilter) FilterTools(ctx context.Context, toolNames []string) []string {
	if len(tf.allowlist) == 0 && len(tf.denylist) == 0 && tf.policyEngine == nil {
		return toolNames // No filtering configured
	}

	filtered := make([]string, 0, len(toolNames))
	for _, name := range toolNames {
		if tf.IsAllowed(ctx, name).IsAllowed() {
			filtered = append(filtered, name)
		}
	}
	return filtered
}

// matchesList checks if toolName matches any pattern in the list.
// Supports glob patterns (e.g., "Bash(*)", "pdf:*").
func (tf *ToolFilter) matchesList(toolName string, list map[string]bool) bool {
	if list[toolName] {
		return true
	}

	// Check glob patterns
	for pattern := range list {
		if ok, err := path.Match(pattern, toolName); err == nil && ok {
			return true
		}
	}

	return false
}

// AddToAllowlist adds tools to the allowlist.
func (tf *ToolFilter) AddToAllowlist(tools ...string) {
	for _, tool := range tools {
		tool = strings.TrimSpace(tool)
		if tool != "" {
			tf.allowlist[tool] = true
		}
	}
}

// AddToDenylist adds tools to the denylist.
func (tf *ToolFilter) AddToDenylist(tools ...string) {
	for _, tool := range tools {
		tool = strings.TrimSpace(tool)
		if tool != "" {
			tf.denylist[tool] = true
		}
	}
}

// AllowlistFromSkills extracts allowed-tools from skill specs and returns them.
// This bridges the AgentSkills allowed-tools field with governance filtering.
func AllowlistFromSkills(allowedTools [][]string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, tools := range allowedTools {
		for _, tool := range tools {
			tool = strings.TrimSpace(tool)
			if tool != "" && !seen[tool] {
				seen[tool] = true
				result = append(result, tool)
			}
		}
	}

	return result
}
