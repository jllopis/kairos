# Guía de Integración: Errores y Observabilidad en Agentes

Esta guía explica cómo usar el sistema de manejo de errores y observabilidad de Kairos en loops de agentes.

---

## Arquitectura de Integración

La observabilidad en Kairos se integra con los agentes en **5 capas**:

```
┌─────────────────────────────────────────────────────────┐
│  Agent Loop                                              │
├─────────────────────────────────────────────────────────┤
│ 1. Trazas OpenTelemetry: tracer.Start()                 │
│ 2. Error Handling: KairosError con contexto             │
│ 3. Métricas: RecordErrorMetric() y RecordRecovery()     │
│ 4. Health Checks: Verificar estado de componentes       │
│ 5. Circuit Breaker: Prevenir cascadas de fallos         │
└─────────────────────────────────────────────────────────┘
```

---

## Puntos de Integración

El agent loop tiene varios puntos donde integrar manejo de errores:

| Punto | Descripción | Error típico |
|-------|-------------|--------------|
| LLM Call | Llamadas al modelo | `CodeLLMError`, `CodeTimeout` |
| Tool Execution | Ejecución de herramientas | `CodeToolFailure` |
| Memory Operations | Lectura/escritura memoria | `CodeMemoryError` |
| Policy Evaluation | Evaluación de políticas | `CodeUnauthorized` |

---

## Integración con Trazas OpenTelemetry

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
)

func (a *Agent) runIteration(ctx context.Context) error {
    tracer := otel.Tracer("kairos.agent")
    
    // Crear span para la iteración
    ctx, span := tracer.Start(ctx, "agent.iteration")
    defer span.End()
    
    // Añadir atributos
    span.SetAttributes(
        attribute.String("agent.id", a.id),
        attribute.Int("iteration", a.currentIteration),
    )
    
    // El error se registra automáticamente en el span
    result, err := a.executeLLM(ctx)
    if err != nil {
        span.RecordError(err)
        return err
    }
    
    return nil
}
```

---

## Manejo de Errores en LLM

```go
import (
    "github.com/jllopis/kairos/pkg/errors"
    "github.com/jllopis/kairos/pkg/agent"
)

func (a *Agent) executeLLM(ctx context.Context) (*Response, error) {
    // Configurar retry para llamadas LLM
    config := errors.RetryConfig{
        MaxAttempts:  3,
        InitialDelay: 500 * time.Millisecond,
        MaxDelay:     10 * time.Second,
        Multiplier:   2.0,
    }
    
    return errors.RetryRecoverable(ctx, config, func() (*Response, error) {
        resp, err := a.llm.Chat(ctx, a.messages)
        if err != nil {
            // Wrap con contexto para observabilidad
            return nil, agent.WrapLLMError(err, "chat completion failed")
        }
        return resp, nil
    })
}
```

---

## Manejo de Errores en Herramientas

```go
func (a *Agent) executeTool(ctx context.Context, call llm.ToolCall) (any, error) {
    tracer := otel.Tracer("kairos.agent")
    ctx, span := tracer.Start(ctx, "tool.execute",
        trace.WithAttributes(
            attribute.String("tool.name", call.Name),
        ),
    )
    defer span.End()
    
    // Ejecutar con timeout
    ctx, cancel := context.WithTimeout(ctx, a.toolTimeout)
    defer cancel()
    
    result, err := a.toolExecutor.Execute(ctx, call.Name, call.Args)
    if err != nil {
        // Wrap error con contexto de la herramienta
        kerr := agent.WrapToolError(err, call.Name, call.Args)
        
        // Registrar métrica de error
        telemetry.RecordErrorMetric(ctx, kerr, span)
        
        // Si es recuperable, intentar fallback
        if errors.IsRecoverable(kerr) {
            return a.tryToolFallback(ctx, call)
        }
        
        return nil, kerr
    }
    
    return result, nil
}

func (a *Agent) tryToolFallback(ctx context.Context, call llm.ToolCall) (any, error) {
    fallback := a.getFallbackTool(call.Name)
    if fallback == "" {
        return nil, errors.New(errors.CodeToolFailure, "no fallback available").
            WithContext("tool", call.Name)
    }
    
    return a.toolExecutor.Execute(ctx, fallback, call.Args)
}
```

---

## Health Checks

Verificar la salud de componentes antes de usarlos:

```go
import "github.com/jllopis/kairos/pkg/agent"

func (a *Agent) checkComponentHealth(ctx context.Context) error {
    // Verificar LLM
    llmHealth := agent.NewLLMHealthChecker(a.llm)
    if status := llmHealth.Check(ctx); status.Status != core.HealthStatusHealthy {
        return errors.New(errors.CodeLLMError, "LLM provider unhealthy").
            WithContext("details", status.Message)
    }
    
    // Verificar memoria si está configurada
    if a.memory != nil {
        memHealth := agent.NewMemoryHealthChecker(a.memory)
        if status := memHealth.Check(ctx); status.Status != core.HealthStatusHealthy {
            // Memoria degradada, pero podemos continuar
            slog.WarnContext(ctx, "memory degraded", "details", status.Message)
        }
    }
    
    return nil
}
```

---

## Circuit Breaker para Servicios Externos

```go
import "github.com/jllopis/kairos/pkg/errors"

// Crear circuit breaker por herramienta
func (a *Agent) initCircuitBreakers() {
    a.toolBreakers = make(map[string]*errors.CircuitBreaker)
    
    config := errors.CircuitBreakerConfig{
        FailureThreshold: 5,
        ResetTimeout:     30 * time.Second,
        HalfOpenRequests: 2,
    }
    
    for _, tool := range a.tools {
        a.toolBreakers[tool.Name] = errors.NewCircuitBreaker(config)
    }
}

func (a *Agent) executeToolWithBreaker(ctx context.Context, name string, args map[string]any) (any, error) {
    breaker, ok := a.toolBreakers[name]
    if !ok {
        return a.toolExecutor.Execute(ctx, name, args)
    }
    
    return breaker.Execute(ctx, func() (any, error) {
        return a.toolExecutor.Execute(ctx, name, args)
    })
}
```

---

## Métricas Automáticas

Las métricas se registran automáticamente cuando usas los helpers:

```go
// Al ocurrir un error
telemetry.RecordErrorMetric(ctx, err, span)
// Incrementa: kairos.errors.total{code="TOOL_FAILURE", tool="get_weather"}

// Tras una recuperación exitosa
telemetry.RecordRecovery(ctx, err)
// Incrementa: kairos.errors.recovered{code="TOOL_FAILURE"}

// Las métricas incluyen automáticamente:
// - error.code
// - error.recoverable
// - component (agent, tool, memory, etc.)
// - tool.name (si aplica)
```

---

## Ejemplo Completo: Agent Loop con Observabilidad

```go
func (a *Agent) Run(ctx context.Context, input string) (string, error) {
    // Health check inicial
    if err := a.checkComponentHealth(ctx); err != nil {
        return "", err
    }
    
    tracer := otel.Tracer("kairos.agent")
    ctx, span := tracer.Start(ctx, "agent.run")
    defer span.End()
    
    a.messages = append(a.messages, llm.Message{Role: "user", Content: input})
    
    for i := 0; i < a.maxIterations; i++ {
        iterCtx, iterSpan := tracer.Start(ctx, "agent.iteration",
            trace.WithAttributes(attribute.Int("iteration", i)),
        )
        
        // Llamar a LLM con retry
        resp, err := a.executeLLM(iterCtx)
        if err != nil {
            iterSpan.RecordError(err)
            iterSpan.End()
            return "", err
        }
        
        // ¿Hay tool calls?
        if len(resp.ToolCalls) == 0 {
            iterSpan.End()
            return resp.Content, nil
        }
        
        // Ejecutar herramientas
        for _, call := range resp.ToolCalls {
            result, err := a.executeToolWithBreaker(iterCtx, call.Name, call.Args)
            if err != nil {
                // Error no recuperable, añadir mensaje de error y continuar
                a.messages = append(a.messages, llm.Message{
                    Role:    "tool",
                    Content: fmt.Sprintf("Error: %s", err),
                    ToolCallID: call.ID,
                })
                continue
            }
            
            a.messages = append(a.messages, llm.Message{
                Role:    "tool",
                Content: fmt.Sprintf("%v", result),
                ToolCallID: call.ID,
            })
        }
        
        iterSpan.End()
    }
    
    return "", errors.New(errors.CodeInternal, "max iterations reached")
}
```

---

## Documentación Relacionada

- **[Manejo de Errores](ERROR_HANDLING.md)** - Tipos de error y retry
- **[Guía de Observabilidad](OBSERVABILITY.md)** - Dashboards y alertas
- **[Exportación de Métricas](METRICS_EXPORT.md)** - Configuración OTLP
