// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"testing"

	"github.com/jllopis/kairos/pkg/config"
)

func TestBuildExplainResult(t *testing.T) {
	cfg := &config.Config{
		LLM: config.LLMConfig{
			Provider: "ollama",
			Model:    "llama3",
			BaseURL:  "http://localhost:11434",
		},
		Memory: config.MemoryConfig{
			Enabled:  true,
			Provider: "inmemory",
		},
		Governance: config.GovernanceConfig{
			Policies: []config.PolicyRuleConfig{
				{ID: "policy1", Effect: "deny"},
				{ID: "policy2", Effect: "allow"},
			},
		},
	}

	result := buildExplainResult(context.Background(), cfg, "test-agent", "")

	if result.Agent != "test-agent" {
		t.Errorf("expected agent id %q, got %q", "test-agent", result.Agent)
	}

	if result.LLM.Provider != "ollama" {
		t.Errorf("expected LLM provider %q, got %q", "ollama", result.LLM.Provider)
	}

	if result.LLM.Model != "llama3" {
		t.Errorf("expected LLM model %q, got %q", "llama3", result.LLM.Model)
	}

	if !result.Memory.Enabled {
		t.Error("expected memory to be enabled")
	}

	if result.Memory.Provider != "inmemory" {
		t.Errorf("expected memory provider %q, got %q", "inmemory", result.Memory.Provider)
	}

	if !result.Governance.Enabled {
		t.Error("expected governance to be enabled")
	}

	if len(result.Governance.Policies) != 2 {
		t.Errorf("expected 2 policies, got %d", len(result.Governance.Policies))
	}
}

func TestExtractPolicyNames(t *testing.T) {
	policies := []config.PolicyRuleConfig{
		{ID: "p1"},
		{ID: "p2"},
		{ID: ""},
	}

	names := extractPolicyNames(policies)

	if len(names) != 2 {
		t.Errorf("expected 2 policy names, got %d", len(names))
	}

	if names[0] != "p1" || names[1] != "p2" {
		t.Errorf("unexpected policy names: %v", names)
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is a long string", 10, "this is..."},
		{"has\nnewline", 15, "has newline"},
	}

	for _, tc := range tests {
		result := truncate(tc.input, tc.maxLen)
		if result != tc.expected {
			t.Errorf("truncate(%q, %d) = %q, expected %q", tc.input, tc.maxLen, result, tc.expected)
		}
	}
}
