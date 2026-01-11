package planner

import "fmt"

// Graph defines a deterministic execution graph for the explicit planner.
type Graph struct {
	ID    string          `json:"id" yaml:"id"`
	Start string          `json:"start" yaml:"start"`
	Nodes map[string]Node `json:"nodes" yaml:"nodes"`
	Edges []Edge          `json:"edges" yaml:"edges"`
}

// Node represents a step in the graph.
type Node struct {
	ID       string            `json:"id" yaml:"id"`
	Type     string            `json:"type" yaml:"type"`
	Tool     string            `json:"tool,omitempty" yaml:"tool,omitempty"`
	Input    any               `json:"input,omitempty" yaml:"input,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// Edge defines a transition between nodes.
type Edge struct {
	From      string `json:"from" yaml:"from"`
	To        string `json:"to" yaml:"to"`
	Condition string `json:"condition,omitempty" yaml:"condition,omitempty"`
}

// Validate ensures the graph is well-formed for execution.
func (g *Graph) Validate() error {
	if g == nil {
		return fmt.Errorf("graph is nil")
	}
	if len(g.Nodes) == 0 {
		return fmt.Errorf("graph has no nodes")
	}

	for id, node := range g.Nodes {
		if node.ID == "" {
			node.ID = id
			g.Nodes[id] = node
		}
		if node.ID == "" {
			return fmt.Errorf("node id is required")
		}
		if node.Type == "" {
			return fmt.Errorf("node %q missing type", node.ID)
		}
	}

	for _, edge := range g.Edges {
		if edge.From == "" || edge.To == "" {
			return fmt.Errorf("edge must include from/to")
		}
		if _, ok := g.Nodes[edge.From]; !ok {
			return fmt.Errorf("edge from %q not found", edge.From)
		}
		if _, ok := g.Nodes[edge.To]; !ok {
			return fmt.Errorf("edge to %q not found", edge.To)
		}
	}
	return nil
}
