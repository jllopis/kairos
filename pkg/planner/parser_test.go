package planner

import "testing"

func TestParseJSON(t *testing.T) {
	payload := []byte(`{
  "id": "graph-json",
  "start": "n1",
  "nodes": {
    "n1": { "type": "noop" },
    "n2": { "type": "noop" }
  },
  "edges": [
    { "from": "n1", "to": "n2" }
  ]
}`)
	graph, err := ParseJSON(payload)
	if err != nil {
		t.Fatalf("parse json: %v", err)
	}
	if graph.ID != "graph-json" {
		t.Fatalf("unexpected graph id: %q", graph.ID)
	}
	if graph.Nodes["n1"].Type != "noop" {
		t.Fatalf("unexpected node type: %q", graph.Nodes["n1"].Type)
	}
}

func TestParseYAML(t *testing.T) {
	payload := []byte(`
id: graph-yaml
start: n1
nodes:
  n1:
    type: noop
  n2:
    type: noop
edges:
  - from: n1
    to: n2
`)
	graph, err := ParseYAML(payload)
	if err != nil {
		t.Fatalf("parse yaml: %v", err)
	}
	if graph.ID != "graph-yaml" {
		t.Fatalf("unexpected graph id: %q", graph.ID)
	}
	if graph.Nodes["n2"].Type != "noop" {
		t.Fatalf("unexpected node type: %q", graph.Nodes["n2"].Type)
	}
}

func TestMarshalRoundTrip(t *testing.T) {
	graph := &Graph{
		ID:    "graph-rt",
		Start: "n1",
		Nodes: map[string]Node{
			"n1": {Type: "noop", Input: "hello"},
			"n2": {Type: "noop", Input: "world"},
		},
		Edges: []Edge{
			{From: "n1", To: "n2"},
		},
	}

	jsonPayload, err := MarshalJSON(graph, true)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	parsedJSON, err := ParseJSON(jsonPayload)
	if err != nil {
		t.Fatalf("parse json: %v", err)
	}
	if parsedJSON.ID != graph.ID {
		t.Fatalf("json round-trip mismatch: %q", parsedJSON.ID)
	}

	yamlPayload, err := MarshalYAML(graph)
	if err != nil {
		t.Fatalf("marshal yaml: %v", err)
	}
	parsedYAML, err := ParseYAML(yamlPayload)
	if err != nil {
		t.Fatalf("parse yaml: %v", err)
	}
	if parsedYAML.ID != graph.ID {
		t.Fatalf("yaml round-trip mismatch: %q", parsedYAML.ID)
	}
}
