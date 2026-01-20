// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package governance

import (
	"context"
	"testing"
)

func TestToolFilter_EmptyFilter(t *testing.T) {
	filter := NewToolFilter()

	decision := filter.IsAllowed(context.Background(), "any-tool")
	if !decision.IsAllowed() {
		t.Error("empty filter should allow all tools")
	}
}

func TestToolFilter_Allowlist(t *testing.T) {
	filter := NewToolFilter(
		WithAllowlist([]string{"allowed-tool", "another-tool"}),
	)

	tests := []struct {
		name    string
		tool    string
		allowed bool
	}{
		{"allowed tool", "allowed-tool", true},
		{"another allowed", "another-tool", true},
		{"not in list", "blocked-tool", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			decision := filter.IsAllowed(context.Background(), tc.tool)
			if decision.IsAllowed() != tc.allowed {
				t.Errorf("tool %q: expected allowed=%v, got %v", tc.tool, tc.allowed, decision.IsAllowed())
			}
		})
	}
}

func TestToolFilter_Denylist(t *testing.T) {
	filter := NewToolFilter(
		WithDenylist([]string{"dangerous-tool"}),
	)

	tests := []struct {
		name    string
		tool    string
		allowed bool
	}{
		{"denied tool", "dangerous-tool", false},
		{"safe tool", "safe-tool", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			decision := filter.IsAllowed(context.Background(), tc.tool)
			if decision.IsAllowed() != tc.allowed {
				t.Errorf("tool %q: expected allowed=%v, got %v", tc.tool, tc.allowed, decision.IsAllowed())
			}
		})
	}
}

func TestToolFilter_DenylistTakesPrecedence(t *testing.T) {
	filter := NewToolFilter(
		WithAllowlist([]string{"tool-a", "tool-b"}),
		WithDenylist([]string{"tool-b"}),
	)

	// tool-a is in allowlist only → allowed
	decision := filter.IsAllowed(context.Background(), "tool-a")
	if !decision.IsAllowed() {
		t.Error("tool-a should be allowed (in allowlist, not in denylist)")
	}

	// tool-b is in both → denied (denylist takes precedence)
	decision = filter.IsAllowed(context.Background(), "tool-b")
	if decision.IsAllowed() {
		t.Error("tool-b should be denied (denylist takes precedence)")
	}
}

func TestToolFilter_GlobPatterns(t *testing.T) {
	filter := NewToolFilter(
		WithAllowlist([]string{"Bash(*)", "Read"}),
	)

	tests := []struct {
		name    string
		tool    string
		allowed bool
	}{
		{"glob match", "Bash(git)", true},
		{"exact match", "Read", true},
		{"no match", "Write", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			decision := filter.IsAllowed(context.Background(), tc.tool)
			if decision.IsAllowed() != tc.allowed {
				t.Errorf("tool %q: expected allowed=%v, got %v", tc.tool, tc.allowed, decision.IsAllowed())
			}
		})
	}
}

func TestToolFilter_FilterTools(t *testing.T) {
	filter := NewToolFilter(
		WithAllowlist([]string{"tool-a", "tool-c"}),
	)

	input := []string{"tool-a", "tool-b", "tool-c", "tool-d"}
	expected := []string{"tool-a", "tool-c"}

	result := filter.FilterTools(context.Background(), input)

	if len(result) != len(expected) {
		t.Fatalf("expected %d tools, got %d", len(expected), len(result))
	}

	for i, name := range expected {
		if result[i] != name {
			t.Errorf("index %d: expected %q, got %q", i, name, result[i])
		}
	}
}

func TestToolFilter_AddToLists(t *testing.T) {
	filter := NewToolFilter()

	filter.AddToAllowlist("new-tool")
	filter.AddToDenylist("bad-tool")

	// new-tool should be allowed
	if !filter.IsAllowed(context.Background(), "new-tool").IsAllowed() {
		t.Error("new-tool should be allowed after AddToAllowlist")
	}

	// bad-tool should be denied
	if filter.IsAllowed(context.Background(), "bad-tool").IsAllowed() {
		t.Error("bad-tool should be denied after AddToDenylist")
	}

	// other tools are not in allowlist, so denied
	if filter.IsAllowed(context.Background(), "other-tool").IsAllowed() {
		t.Error("other-tool should be denied (not in allowlist)")
	}
}

func TestToolFilter_WithPolicyEngine(t *testing.T) {
	// Create a simple policy engine that denies "policy-denied"
	ruleSet := NewRuleSet([]Rule{
		{ID: "deny-specific", Effect: "deny", Type: ActionTool, Name: "policy-denied"},
	})

	filter := NewToolFilter(
		WithPolicyEngine(ruleSet),
	)

	// Regular tool should be allowed (default allow)
	if !filter.IsAllowed(context.Background(), "regular-tool").IsAllowed() {
		t.Error("regular-tool should be allowed by policy")
	}

	// Denied by policy
	if filter.IsAllowed(context.Background(), "policy-denied").IsAllowed() {
		t.Error("policy-denied should be denied by policy engine")
	}
}

func TestAllowlistFromSkills(t *testing.T) {
	allowedTools := [][]string{
		{"Bash(git:*)", "Read"},
		{"Bash(jq:*)", "Read"}, // Read is duplicate
		{"Write"},
	}

	result := AllowlistFromSkills(allowedTools)

	expected := map[string]bool{
		"Bash(git:*)": true,
		"Read":        true,
		"Bash(jq:*)":  true,
		"Write":       true,
	}

	if len(result) != len(expected) {
		t.Errorf("expected %d tools, got %d: %v", len(expected), len(result), result)
	}

	for _, tool := range result {
		if !expected[tool] {
			t.Errorf("unexpected tool: %q", tool)
		}
	}
}
