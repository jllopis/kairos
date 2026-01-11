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

## Branching conditions

Use `condition` in edges to select the next node. Supported conditions:
- `last==<value>` / `last!=<value>`
- `output.<node_id>==<value>` / `output.<node_id>!=<value>`
- `default` (fallback when no condition matches)

Example:

```yaml
nodes:
  step-1:
    type: log
    input: "ok"
  step-2:
    type: log
    input: "branch-ok"
  step-3:
    type: log
    input: "branch-default"
edges:
  - from: step-1
    to: step-2
    condition: "last==ok"
  - from: step-1
    to: step-3
    condition: "default"
```

## Serialization helpers

```go
jsonPayload, err := planner.MarshalJSON(graph, true)
yamlPayload, err := planner.MarshalYAML(graph)
```

## Audit hook

```go
exec := planner.NewExecutor(handlers)
exec.AuditHook = func(ctx context.Context, event planner.AuditEvent) {
  fmt.Printf("node=%s status=%s\n", event.NodeID, event.Status)
}
```

## Notes
- Branching uses first matching condition and an optional `default` edge.
- Audit events emit start/completed/failed with timestamps and output.
