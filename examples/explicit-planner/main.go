package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jllopis/kairos/pkg/planner"
)

func main() {
	payload := []byte(`
id: example-graph
start: step-1
nodes:
  step-1:
    type: log
    input: "hello"
  step-2:
    type: log
    input: "world"
edges:
  - from: step-1
    to: step-2
`)

	graph, err := planner.ParseYAML(payload)
	if err != nil {
		log.Fatalf("failed to parse graph: %v", err)
	}

	exec := planner.NewExecutor(map[string]planner.Handler{
		"log": func(_ context.Context, node planner.Node, _ *planner.State) (any, error) {
			fmt.Printf("node=%s input=%v\n", node.ID, node.Input)
			return node.Input, nil
		},
	})

	state, err := exec.Execute(context.Background(), graph, nil)
	if err != nil {
		log.Fatalf("execution failed: %v", err)
	}

	fmt.Printf("last output: %v\n", state.Last)
}
