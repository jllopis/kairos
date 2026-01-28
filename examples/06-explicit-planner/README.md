# 06 - Explicit Planner

Orquestación determinista con grafos (DAG) en lugar de autonomía del LLM.

## Qué aprenderás

- Definir un plan como grafo dirigido (DAG)
- Ejecutar pasos en orden con handlers
- Control total del flujo vs. autonomía del agente
- Cuándo usar planner explícito vs. ReAct

## Ejecutar

```bash
go run .
```

## Definición del grafo (YAML)

```yaml
# plan.yaml
id: example-graph
start: start
nodes:
  start:
    type: init
  validate:
    type: validation
  process:
    type: llm_call
  format:
    type: format_output
  error:
    type: error_handler
  end:
    type: terminal
edges:
  - from: start
    to: validate
  - from: validate
    to: process
    condition: "last==ok"
  - from: validate
    to: error
    condition: "default"
  - from: process
    to: format
  - from: format
    to: end
  - from: error
    to: end
```

## Código clave

```go
// Cargar grafo
graph, _ := planner.LoadGraph("plan.yaml")

// Definir handlers para cada tipo de nodo
handlers := map[string]planner.Handler{
    "init":          handleInit,
    "validation":    handleValidation,
    "llm_call":      handleLLMCall,
    "format_output": handleFormat,
    "error_handler": handleError,
    "terminal":      handleEnd,
}

// Ejecutor
executor := planner.NewExecutor(handlers)

// Ejecutar el plan
state := planner.NewState()
result, _ := executor.Execute(ctx, graph, state)
```

### Override por node.id (opt-in)

Si necesitas un handler específico para un nodo concreto, puedes sobrescribirlo
por `node.id` sin cambiar el `type`:

```go
executor.HandlersByID = map[string]planner.Handler{
    "format": func(ctx context.Context, node planner.Node, state *planner.State) (any, error) {
        return "override", nil
    },
}
```

La resolución es: `node.id` (si existe) → `node.type`.

## Cuándo usar planner explícito

| Escenario | Usar |
|-----------|------|
| Flujo predecible y auditable | ✅ Planner |
| Compliance estricto | ✅ Planner |
| Tareas exploratorias | ❌ ReAct |
| Conversación libre | ❌ ReAct |

## Siguiente paso

→ [07-multi-agent-mcp](../07-multi-agent-mcp/) para comunicación entre agentes
