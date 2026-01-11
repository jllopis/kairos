# Walkthrough: Explicit Planner (Graph)

This walkthrough shows how to define a graph, parse YAML/JSON, and execute it with the explicit planner.

## Graph Schema (minimum)

```yaml
id: hello-graph
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
```

Fields:
- `id`: graph identifier (optional).
- `start`: start node id (optional if the graph has a single node with no incoming edges).
- `nodes`: map of node id to node definition.
- `edges`: list of transitions between nodes.

## Parse YAML / JSON

```go
graph, err := planner.ParseYAML(yamlPayload)
if err != nil {
  // handle error
}
```

```go
graph, err := planner.ParseJSON(jsonPayload)
if err != nil {
  // handle error
}
```

## Execute the Graph

```go
exec := planner.NewExecutor(map[string]planner.Handler{
  "log": func(ctx context.Context, node planner.Node, state *planner.State) (any, error) {
    fmt.Printf("node %s input=%v\n", node.ID, node.Input)
    return node.Input, nil
  },
})

state, err := exec.Execute(context.Background(), graph, nil)
if err != nil {
  // handle error
}

fmt.Printf("last output: %v\n", state.Last)
```

## Notes
- The executor currently supports a single linear path (one outgoing edge per node).
- Branching/conditions and richer audit events are planned in a later phase.
