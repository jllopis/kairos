// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"testing"

	"github.com/jllopis/kairos/pkg/planner"
)

func TestToMermaid(t *testing.T) {
	graph := &planner.Graph{
		ID:    "test-graph",
		Start: "a",
		Nodes: map[string]planner.Node{
			"a": {ID: "a", Type: "agent"},
			"b": {ID: "b", Type: "tool"},
		},
		Edges: []planner.Edge{
			{From: "a", To: "b"},
		},
	}

	result := toMermaid(graph)

	if result == "" {
		t.Error("expected non-empty mermaid output")
	}

	// Should contain graph directive
	if !contains(result, "graph TD") {
		t.Error("expected graph TD directive")
	}

	// Should contain nodes
	if !contains(result, "a[a: agent]") {
		t.Error("expected node a")
	}

	// Should contain edge
	if !contains(result, "a --> b") {
		t.Error("expected edge a --> b")
	}

	// Should highlight start node
	if !contains(result, "style a fill:#90EE90") {
		t.Error("expected start node highlight")
	}
}

func TestToDot(t *testing.T) {
	graph := &planner.Graph{
		ID:    "test-graph",
		Start: "start",
		Nodes: map[string]planner.Node{
			"start": {ID: "start", Type: "entry"},
			"end":   {ID: "end", Type: "exit"},
		},
		Edges: []planner.Edge{
			{From: "start", To: "end", Condition: "always"},
		},
	}

	result := toDot(graph)

	if result == "" {
		t.Error("expected non-empty dot output")
	}

	// Should contain digraph
	if !contains(result, "digraph G") {
		t.Error("expected digraph G")
	}

	// Should contain node with filled style for start
	if !contains(result, "fillcolor=\"#90EE90\"") {
		t.Error("expected start node fillcolor")
	}

	// Should contain edge
	if !contains(result, "\"start\" -> \"end\"") {
		t.Error("expected edge start -> end")
	}
}

func TestToMermaidWithCondition(t *testing.T) {
	graph := &planner.Graph{
		ID: "conditional-graph",
		Nodes: map[string]planner.Node{
			"check": {ID: "check", Type: "decision"},
			"yes":   {ID: "yes", Type: "action"},
			"no":    {ID: "no", Type: "action"},
		},
		Edges: []planner.Edge{
			{From: "check", To: "yes", Condition: "last==true"},
			{From: "check", To: "no", Condition: "last==false"},
		},
	}

	result := toMermaid(graph)

	// Should contain conditional edges with labels
	if !contains(result, "check -->|last==true| yes") {
		t.Error("expected conditional edge with label")
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsInner(s, substr)))
}

func containsInner(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
