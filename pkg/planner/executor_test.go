package planner

import (
	"context"
	"testing"
)

func TestExecutorSinglePath(t *testing.T) {
	graph := &Graph{
		ID:    "graph",
		Start: "n1",
		Nodes: map[string]Node{
			"n1": {Type: "noop", Input: "first"},
			"n2": {Type: "noop", Input: "second"},
		},
		Edges: []Edge{
			{From: "n1", To: "n2"},
		},
	}

	exec := NewExecutor(map[string]Handler{
		"noop": func(_ context.Context, node Node, _ *State) (any, error) {
			return node.Input, nil
		},
	})

	state, err := exec.Execute(context.Background(), graph, nil)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if state.Last != "second" {
		t.Fatalf("unexpected last output: %v", state.Last)
	}
	if state.Outputs["n1"] != "first" {
		t.Fatalf("unexpected n1 output: %v", state.Outputs["n1"])
	}
	if state.Outputs["n2"] != "second" {
		t.Fatalf("unexpected n2 output: %v", state.Outputs["n2"])
	}
}

func TestExecutorBranching(t *testing.T) {
	graph := &Graph{
		ID:    "graph-branch",
		Start: "n1",
		Nodes: map[string]Node{
			"n1": {Type: "noop", Input: "ok"},
			"n2": {Type: "noop", Input: "branch-ok"},
			"n3": {Type: "noop", Input: "branch-default"},
		},
		Edges: []Edge{
			{From: "n1", To: "n2", Condition: "last==ok"},
			{From: "n1", To: "n3", Condition: "default"},
		},
	}

	exec := NewExecutor(map[string]Handler{
		"noop": func(_ context.Context, node Node, _ *State) (any, error) {
			return node.Input, nil
		},
	})

	state, err := exec.Execute(context.Background(), graph, nil)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if state.Last != "branch-ok" {
		t.Fatalf("unexpected last output: %v", state.Last)
	}
}

func TestExecutorAuditHook(t *testing.T) {
	graph := &Graph{
		ID:    "graph-audit",
		Start: "n1",
		Nodes: map[string]Node{
			"n1": {Type: "noop", Input: "one"},
		},
	}

	var events []AuditEvent
	exec := NewExecutor(map[string]Handler{
		"noop": func(_ context.Context, node Node, _ *State) (any, error) {
			return node.Input, nil
		},
	})
	exec.AuditHook = func(_ context.Context, event AuditEvent) {
		events = append(events, event)
	}

	_, err := exec.Execute(context.Background(), graph, nil)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 audit events, got %d", len(events))
	}
	if events[0].Status != "started" || events[1].Status != "completed" {
		t.Fatalf("unexpected audit statuses: %+v", events)
	}
}

func TestExecutorHandlersByIDOverridesType(t *testing.T) {
	graph := &Graph{
		ID:    "graph-id",
		Start: "n1",
		Nodes: map[string]Node{
			"n1": {Type: "noop", Input: "type-1"},
			"n2": {Type: "noop", Input: "type-2"},
		},
		Edges: []Edge{
			{From: "n1", To: "n2"},
		},
	}

	exec := NewExecutor(map[string]Handler{
		"noop": func(_ context.Context, node Node, _ *State) (any, error) {
			return node.Input, nil
		},
	})
	exec.HandlersByID = map[string]Handler{
		"n1": func(_ context.Context, _ Node, _ *State) (any, error) {
			return "id-override", nil
		},
	}

	state, err := exec.Execute(context.Background(), graph, nil)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if state.Outputs["n1"] != "id-override" {
		t.Fatalf("expected id override output, got: %v", state.Outputs["n1"])
	}
	if state.Outputs["n2"] != "type-2" {
		t.Fatalf("expected type handler output, got: %v", state.Outputs["n2"])
	}
}
