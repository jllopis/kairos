package planner_test

import (
	"context"
	"fmt"

	"github.com/jllopis/kairos/pkg/planner"
)

func ExampleExecutor() {
	graph := &planner.Graph{
		Start: "start",
		Nodes: map[string]planner.Node{
			"start": {ID: "start", Type: "echo", Input: "hello"},
			"end":   {ID: "end", Type: "echo", Input: "done"},
		},
		Edges: []planner.Edge{{From: "start", To: "end"}},
	}

	executor := planner.NewExecutor(map[string]planner.Handler{
		"echo": func(_ context.Context, node planner.Node, _ *planner.State) (any, error) {
			return node.Input, nil
		},
	})

	state, err := executor.Execute(context.Background(), graph, nil)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(state.Last)
	// Output: done
}
