package planner

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Handler executes a node and can update state.
type Handler func(ctx context.Context, node Node, state *State) (any, error)

// State holds outputs produced during graph execution.
type State struct {
	Last    any
	Outputs map[string]any
}

// NewState creates an initialized execution state.
func NewState() *State {
	return &State{Outputs: make(map[string]any)}
}

// Executor runs a graph using node handlers.
type Executor struct {
	Handlers map[string]Handler
	tracer   trace.Tracer
}

// NewExecutor creates an executor with provided handlers.
func NewExecutor(handlers map[string]Handler) *Executor {
	return &Executor{
		Handlers: handlers,
		tracer:   otel.Tracer("kairos/planner"),
	}
}

// Execute runs the graph from its start node and returns the final state.
func (e *Executor) Execute(ctx context.Context, graph *Graph, state *State) (*State, error) {
	if graph == nil {
		return nil, fmt.Errorf("graph is nil")
	}
	if err := graph.Validate(); err != nil {
		return nil, err
	}
	if state == nil {
		state = NewState()
	}

	startID, err := resolveStartNode(graph)
	if err != nil {
		return nil, err
	}

	adjacency, err := buildAdjacency(graph)
	if err != nil {
		return nil, err
	}

	visited := make(map[string]bool)
	currentID := startID
	for currentID != "" {
		if visited[currentID] {
			return nil, fmt.Errorf("cycle detected at node %q", currentID)
		}
		visited[currentID] = true

		node, ok := graph.Nodes[currentID]
		if !ok {
			return nil, fmt.Errorf("node %q not found", currentID)
		}

		handler := e.Handlers[node.Type]
		if handler == nil {
			return nil, fmt.Errorf("no handler for node type %q", node.Type)
		}

		nodeCtx, span := e.tracer.Start(ctx, "Planner.Node",
			trace.WithAttributes(
				attribute.String("node.id", node.ID),
				attribute.String("node.type", node.Type),
			),
		)
		output, err := handler(nodeCtx, node, state)
		span.End()
		if err != nil {
			return nil, fmt.Errorf("node %q failed: %w", node.ID, err)
		}
		state.Outputs[node.ID] = output
		state.Last = output

		next := adjacency[currentID]
		if len(next) == 0 {
			break
		}
		if len(next) > 1 {
			return nil, fmt.Errorf("node %q has multiple outgoing edges", currentID)
		}
		currentID = next[0]
	}

	return state, nil
}

func resolveStartNode(graph *Graph) (string, error) {
	if graph.Start != "" {
		if _, ok := graph.Nodes[graph.Start]; !ok {
			return "", fmt.Errorf("start node %q not found", graph.Start)
		}
		return graph.Start, nil
	}

	incoming := make(map[string]int)
	for id := range graph.Nodes {
		incoming[id] = 0
	}
	for _, edge := range graph.Edges {
		incoming[edge.To]++
	}

	var candidates []string
	for id, count := range incoming {
		if count == 0 {
			candidates = append(candidates, id)
		}
	}
	if len(candidates) == 1 {
		return candidates[0], nil
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("no start node found")
	}
	return "", fmt.Errorf("multiple start nodes found")
}

func buildAdjacency(graph *Graph) (map[string][]string, error) {
	adj := make(map[string][]string, len(graph.Nodes))
	for id := range graph.Nodes {
		adj[id] = nil
	}
	for _, edge := range graph.Edges {
		if edge.From == "" || edge.To == "" {
			return nil, fmt.Errorf("edge must include from/to")
		}
		adj[edge.From] = append(adj[edge.From], edge.To)
	}
	return adj, nil
}
