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
