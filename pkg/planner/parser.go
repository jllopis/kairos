package planner

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

// ParseJSON loads a graph from JSON and validates it.
func ParseJSON(data []byte) (*Graph, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty JSON payload")
	}
	var graph Graph
	if err := json.Unmarshal(data, &graph); err != nil {
		return nil, fmt.Errorf("parse json graph: %w", err)
	}
	if err := graph.Validate(); err != nil {
		return nil, err
	}
	return &graph, nil
}

// ParseYAML loads a graph from YAML and validates it.
func ParseYAML(data []byte) (*Graph, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty YAML payload")
	}
	var graph Graph
	if err := yaml.Unmarshal(data, &graph); err != nil {
		return nil, fmt.Errorf("parse yaml graph: %w", err)
	}
	if err := graph.Validate(); err != nil {
		return nil, err
	}
	return &graph, nil
}

// MarshalJSON serializes a graph to JSON. Use pretty for indented output.
func MarshalJSON(graph *Graph, pretty bool) ([]byte, error) {
	if graph == nil {
		return nil, fmt.Errorf("graph is nil")
	}
	if err := graph.Validate(); err != nil {
		return nil, err
	}
	if pretty {
		return json.MarshalIndent(graph, "", "  ")
	}
	return json.Marshal(graph)
}

// MarshalYAML serializes a graph to YAML.
func MarshalYAML(graph *Graph) ([]byte, error) {
	if graph == nil {
		return nil, fmt.Errorf("graph is nil")
	}
	if err := graph.Validate(); err != nil {
		return nil, err
	}
	return yaml.Marshal(graph)
}
