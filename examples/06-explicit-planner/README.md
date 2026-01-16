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
nodes:
  - id: start
    type: init
    next: [validate]
    
  - id: validate
    type: validation
    next: [process, error]
    
  - id: process
    type: llm_call
    next: [format]
    
  - id: format
    type: format_output
    next: [end]
    
  - id: error
    type: error_handler
    next: [end]
    
  - id: end
    type: terminal
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
result, _ := executor.Run(ctx, graph, state)
```

## Cuándo usar planner explícito

| Escenario | Usar |
|-----------|------|
| Flujo predecible y auditable | ✅ Planner |
| Compliance estricto | ✅ Planner |
| Tareas exploratorias | ❌ ReAct |
| Conversación libre | ❌ ReAct |

## Siguiente paso

→ [07-multi-agent-mcp](../07-multi-agent-mcp/) para comunicación entre agentes
