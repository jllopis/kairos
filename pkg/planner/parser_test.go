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
