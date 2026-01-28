# Planner explícito

El planner explícito permite definir flujos deterministas mediante grafos. Cada
nodo tiene un tipo, un input y conexiones a otros nodos. El grafo puede
serializarse en YAML o JSON y ejecutarse con un motor que conoce los tipos de
nodo.

## Estructura básica

Un grafo tiene un `start` y un conjunto de `nodes`, con `edges` que conectan las
transiciones. Si solo hay un nodo sin entradas, el `start` puede omitirse.

## Ejecución

El executor recorre el grafo y llama a un handler por tipo de nodo. El estado
permite leer la última salida (`state.Last`) y outputs por nodo para tomar
siguientes pasos.

Ejemplo mínimo en YAML:

```yaml
id: hello-graph
start: step-1
nodes:
  step-1:
    type: log
    input: "hola"
  step-2:
    type: log
    input: "mundo"
edges:
  - from: step-1
    to: step-2
```

Ejemplo de ejecución en Go:

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

## Integración con el runtime del agente

El planner explícito se puede ejecutar dentro del runtime del agente:

```go
graph, _ := planner.ParseYAML(planBytes)
ag, _ := agent.New("my-agent", provider, agent.WithPlanner(graph))
out, _ := ag.Run(ctx, "input inicial")
```

Desde CLI:

```bash
kairos run --plan ./plan.yaml --prompt "input inicial"
```

### Tipos de nodo por defecto

- `tool`: ejecuta una herramienta usando `node.tool` o `metadata.tool`.
- `agent`: ejecuta el loop emergente del agente con el `input` del nodo.
- `llm`: llamada directa al LLM sin tool calling.
- `decision` / `noop`: no-op (mantiene `state.Last`).
- Si `node.type` coincide con el nombre de una tool, se ejecuta esa tool.

### Entrada y estado

- Si `input` no está definido, se usa `state.Last`.
- El input inicial está disponible como `state.Outputs["input"]`.
- Si hay memoria configurada, su contexto se expone en `state.Outputs["memory"]`.

## Branching

Las transiciones pueden tener condiciones. Se evalúa la primera que encaje y se
usa `default` como fallback.

Ejemplo:

```yaml
edges:
  - from: step-1
    to: step-2
    condition: "last==ok"
  - from: step-1
    to: step-3
    condition: "default"
```

## Auditoría

El executor puede emitir eventos de auditoría al inicio, fin o fallo de cada
nodo, con timestamps y salida producida.

Ejemplo de hook:

```go
exec.AuditHook = func(ctx context.Context, event planner.AuditEvent) {
  fmt.Printf("node=%s status=%s\n", event.NodeID, event.Status)
}
```
