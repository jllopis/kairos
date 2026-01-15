# SPDX-License-Identifier: Apache-2.0

# Observability Integration Guide: Using Error Handling & Metrics with Agents

> How to leverage Kairos error handling and observability components in agent loops  
> For developers integrating error handling into existing agent code

---

## Quick Answer

La observabilidad en Kairos se integra con los agentes en **5 capas**:

```
┌─────────────────────────────────────────────────────────┐
│  Agent Loop (pkg/agent/agent.go)                        │
├─────────────────────────────────────────────────────────┤
│ 1. Trazas OpenTelemetry: ctx.Tracer().Start()          │
│ 2. Error Handling: KairosError con context              │
│ 3. Métricas: RecordErrorMetric() y RecordRecovery()    │
│ 4. Health Checks: Monitorear estado de components      │
│ 5. Circuit Breaker: Prevenir cascadas de fallos         │
└─────────────────────────────────────────────────────────┘
```

---

## Current State: Agent Loop Architecture

El agent loop actual (ReAct pattern) ya tiene observabilidad básica:

```go
// pkg/agent/agent.go - Main loop (línea 375)
for i := 0; i < a.maxIterations; i++ {
    // 1. Structured logging con trace/span IDs
    slog.InfoContext(ctx, "starting iteration",
        "run_id", a.runID,
        "agent_id", a.id,
        "iteration", i,
    )
    
    // 2. Trazas OpenTelemetry (ya integradas)
    _, llmSpan := tracer.Start(ctx, "llm.chat")
    defer llmSpan.End()
    
    // 3. Métricas básicas
    llmLatencyMs := time.Since(startTime).Milliseconds()
    a.metrics.RecordInt64(ctx, a.llmLatencyMs, llmLatencyMs)
    
    // ❌ Falta: Error handling tipado
    // ❌ Falta: Retry automático
    // ❌ Falta: Health checks
}
```

### Puntos de Integración Existentes

1. **LLM Calls** (línea 380): Donde fallos son comunes
2. **Tool Execution** (línea 575): Donde pueden fallar herramientas
3. **Memory Operations** (línea 365): Donde puede fallar persistencia
4. **Policy Evaluation** (línea 558): Donde pueden estar condiciones
5. **Loop Termination** (línea 638): Donde registrar razón de salida

---

## Phase 4: Integration Pattern (Planned)

La **Fase 4** de implementación migra el agent loop existente para usar los componentes new:

### Antes (Actual)

```go
resp, err := a.llm.Chat(ctx, messages)
if err != nil {
    slog.ErrorContext(ctx, "llm error", "error", err)
    continue // Generic error handling
}
```

### Después (Fase 4)

```go
resp, err := resilience.Retry(ctx, retryConfig, func() (*Response, error) {
    return a.llm.Chat(ctx, messages)
})

// 1. Error automáticamente clasificado
if err != nil {
    kairoErr := err.(*errors.KairosError)
    
    // 2. Contexto rico en traza
    span.SetAttributes(
        attribute.String("error.code", kairoErr.RecoverableString()),
        attribute.String("error.context", kairoErr.ContextKey()),
    )
    
    // 3. Métrica registrada automáticamente
    metrics.RecordErrorMetric(ctx, kairoErr, span)
    
    // 4. Decisión de recuperación
    if kairoErr.Recoverable {
        metrics.RecordRecovery(ctx, kairoErr) // kairos.errors.recovered++
        continue // Ya reintentado
    } else {
        slog.ErrorContext(ctx, "unrecoverable error", "code", kairoErr.Code)
        break
    }
}
```

---

## Integration Points: 5 Capas

### Capa 1: Trazas OpenTelemetry (Existente)

**Ya implementado**, pero mejorable:

```go
// El agent loop ya crea spans
ctx, span := tracer.Start(ctx, "agent.run")
defer span.End()

// LLM calls
_, llmSpan := tracer.Start(ctx, "llm.chat")
defer llmSpan.End()

// Tool execution
_, toolSpan := tracer.Start(ctx, "tool.execute", 
    trace.WithAttributes(
        attribute.String("tool.name", toolName),
    ),
)
defer toolSpan.End()
```

**Mejora Fase 4**: Agregar error context a spans

```go
if err != nil {
    kairoErr := errors.WrapError(err, errors.CodeToolFailure)
    
    // Contexto automáticamente agregado a span
    span.SetAttributes(
        attribute.String("error.code", kairoErr.Code.String()),
        attribute.Bool("error.recoverable", kairoErr.Recoverable),
    )
}
```

---

### Capa 2: Error Handling Tipado (Fase 1-3, Listo)

**Disponible ahora**:

```go
import "github.com/jllopis/kairos/pkg/errors"

// Envolver errores genéricos
err := a.llm.Chat(ctx, messages)
if err != nil {
    // Opción A: Clasificar automáticamente
    kairoErr := errors.FromError(err)
    
    // Opción B: Especificar código conocido
    kairoErr := &errors.KairosError{
        Code: errors.CodeLLMError,
        Msg: "LLM endpoint timeout",
        Recoverable: true,
        Err: err,
    }
    
    return kairoErr
}
```

**9 Códigos disponibles**:
- `CodeTimeout` - Timeout (recuperable)
- `CodeRateLimit` - Rate limiting (recuperable con backoff)
- `CodeToolFailure` - Tool fallido (puede ser recuperable)
- `CodeNotFound` - Recurso no existe (no recuperable)
- `CodeUnauthorized` - Autenticación/permiso (no recuperable)
- `CodeMemoryError` - Memory exhausted (no recuperable)
- `CodeLLMError` - LLM error (variable)
- `CodeInvalidInput` - Input validation (no recuperable)
- `CodeInternal` - Error interno (no recuperable)

---

### Capa 3: Retry & Circuit Breaker (Fase 1, Listo)

**Usar para operaciones propensas a fallos**:

```go
import "github.com/jllopis/kairos/pkg/resilience"

// LLM calls con retry automático
retryConfig := resilience.DefaultRetryConfig().
    WithMaxAttempts(3).
    WithInitialDelay(100 * time.Millisecond).
    WithBackoffMultiplier(2.0) // 100ms, 200ms, 400ms

resp, err := resilience.Retry(ctx, retryConfig, func() (*Response, error) {
    return a.llm.Chat(ctx, messages)
})

// Circuit breaker para evitar hammering servicios muertos
cb := resilience.NewCircuitBreaker(
    resilience.DefaultCircuitBreakerConfig().
        WithThreshold(5).           // Abrir después de 5 fallos
        WithTimeout(30 * time.Second), // Retry en 30s
)

resp, err := cb.Call(ctx, func() (interface{}, error) {
    return a.llm.Chat(ctx, messages)
})
```

**Dónde aplicar en agent loop**:
- ✅ LLM calls (propensas a timeouts)
- ✅ Tool execution (tools pueden no responder)
- ✅ Memory operations (backend puede estar degradado)
- ✅ External API calls (network unreliable)

---

### Capa 4: Health Checks (Fase 2, Listo)

**Monitorear estado de componentes críticos**:

```go
import (
    "github.com/jllopis/kairos/pkg/core"
    "github.com/jllopis/kairos/pkg/telemetry"
)

// Setup health checks al iniciar agent
provider := core.NewDefaultHealthCheckProvider(10 * time.Second)

// Registrar checkers
provider.RegisterChecker("llm-service", func(ctx context.Context) core.HealthStatus {
    _, err := a.llm.Health(ctx)
    if err != nil {
        return core.HealthStatusUnhealthy
    }
    return core.HealthStatusHealthy
})

provider.RegisterChecker("memory-backend", func(ctx context.Context) core.HealthStatus {
    _, err := a.memory.Health(ctx)
    if err != nil {
        return core.HealthStatusDegraded
    }
    return core.HealthStatusHealthy
})

// En el agent loop: check health antes de operaciones críticas
if status := provider.Health(ctx, "llm-service"); status != core.HealthStatusHealthy {
    slog.WarnContext(ctx, "llm service degraded", "status", status)
    // Usar fallback o pausar
}

// Registrar health metrics
status := provider.Health(ctx, "llm-service")
telemetry.RecordHealthStatus(ctx, "llm-service", status)
```

---

### Capa 5: Fallback Strategies (Fase 2, Listo)

**Degradación grácil cuando componentes fallan**:

```go
import "github.com/jllopis/kairos/pkg/resilience"

// Estrategia 1: Static fallback
fallback := resilience.NewStaticFallback(
    func(ctx context.Context) (interface{}, error) {
        return a.llm.Chat(ctx, messages)
    },
    func() (interface{}, error) {
        // Fallback: Usar modelo más simple
        return a.llmFallback.Chat(ctx, messages)
    },
)
resp, err := fallback.Call(ctx)

// Estrategia 2: Cached fallback
fallback := resilience.NewCachedFallback(
    func(ctx context.Context) (interface{}, error) {
        return a.llm.Chat(ctx, messages)
    },
    cache, // Usar últimos resultados si LLM falla
)
resp, err := fallback.Call(ctx)

// Estrategia 3: Chained fallback (try multiple strategies)
fallback := resilience.NewChainedFallback(
    []resilience.FallbackStrategy{
        primaryFallback,
        cachedFallback,
        simplifiedFallback,
    },
)
resp, err := fallback.Call(ctx)
```

---

## Integration Example: Complete Agent Loop

Aquí cómo integrar todas las capas en el agent loop:

```go
package agent

import (
    "context"
    "github.com/jllopis/kairos/pkg/errors"
    "github.com/jllopis/kairos/pkg/resilience"
    "github.com/jllopis/kairos/pkg/core"
    "github.com/jllopis/kairos/pkg/telemetry"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "log/slog"
)

// Enhanced agent loop with observability (Phase 4)
func (a *Agent) runWithObservability(ctx context.Context) error {
    tracer := otel.Tracer("kairos.agent")
    ctx, mainSpan := tracer.Start(ctx, "agent.run")
    defer mainSpan.End()
    
    metrics, err := telemetry.NewErrorMetrics(ctx)
    if err != nil {
        return err
    }
    
    healthProvider := core.NewDefaultHealthCheckProvider(10 * time.Second)
    a.setupHealthChecks(ctx, healthProvider)
    
    retryConfig := resilience.DefaultRetryConfig().
        WithMaxAttempts(3).
        WithInitialDelay(100 * time.Millisecond)
    
    // CAPA 5: Fallback para LLM
    llmFallback := resilience.NewChainedFallback([]resilience.FallbackStrategy{
        resilience.NewStaticFallback(
            func(ctx context.Context) (interface{}, error) {
                return a.llm.Chat(ctx, a.messages)
            },
            func() (interface{}, error) {
                return a.llmFallback.Chat(ctx, a.messages)
            },
        ),
    })
    
    // CAPA 1: Trazas (ya existente)
    // CAPA 4: Health checks
    for i := 0; i < a.maxIterations; i++ {
        slog.InfoContext(ctx, "iteration", "num", i)
        _, iterSpan := tracer.Start(ctx, "agent.iteration",
            trace.WithAttributes(attribute.Int("iteration", i)),
        )
        
        // Check health antes de LLM call
        if status := healthProvider.Health(ctx, "llm-service"); 
            status != core.HealthStatusHealthy {
            slog.WarnContext(ctx, "llm degraded", "status", status)
            metrics.RecordHealthStatus(ctx, "llm-service", status)
            // Continuar con fallback
        }
        
        // CAPA 3: Retry + CAPA 2: Error handling
        resp, err := resilience.Retry(ctx, retryConfig, func() (interface{}, error) {
            return llmFallback.Call(ctx)
        })
        
        if err != nil {
            // CAPA 2: Clasificar error
            kairoErr := errors.WrapError(err, errors.CodeLLMError)
            
            // CAPA 1: Agregar contexto a span
            iterSpan.SetAttributes(
                attribute.String("error.code", kairoErr.Code.String()),
                attribute.Bool("error.recoverable", kairoErr.Recoverable),
            )
            
            // CAPA 3: Métrica
            metrics.RecordErrorMetric(ctx, kairoErr, iterSpan)
            
            if !kairoErr.Recoverable {
                slog.ErrorContext(ctx, "unrecoverable llm error", 
                    "code", kairoErr.Code)
                iterSpan.End()
                break
            }
            
            metrics.RecordRecovery(ctx, kairoErr)
            iterSpan.End()
            continue
        }
        
        // Success path
        chatResp := resp.(*ChatResponse)
        
        // Process response (tool calls, final answer, etc.)
        if err := a.processResponse(ctx, chatResp); err != nil {
            kairoErr := errors.WrapError(err, errors.CodeToolFailure)
            metrics.RecordErrorMetric(ctx, kairoErr, iterSpan)
            iterSpan.End()
            continue
        }
        
        iterSpan.End()
    }
    
    return nil
}

func (a *Agent) setupHealthChecks(ctx context.Context, 
    provider core.HealthCheckProvider) {
    
    provider.RegisterChecker("llm-service", func(ctx context.Context) core.HealthStatus {
        _, err := a.llm.Health(ctx)
        if err != nil {
            return core.HealthStatusUnhealthy
        }
        return core.HealthStatusHealthy
    })
    
    provider.RegisterChecker("memory", func(ctx context.Context) core.HealthStatus {
        _, err := a.memory.Health(ctx)
        if err != nil {
            return core.HealthStatusDegraded
        }
        return core.HealthStatusHealthy
    })
}
```

---

## Integration Roadmap: Phase 4

Aquí el plan concreto para Fase 4:

### Paso 1: Wrap Errors (1-2 horas)
```
- [ ] Identificar todos los `if err != nil` en agent.go
- [ ] Reemplazar con errors.WrapError(..., código) o KairosError{}
- [ ] Asegurar cada error tiene el KairosError.Recoverable correcto
```

### Paso 2: Add Retry Logic (2-3 horas)
```
- [ ] LLM calls wrapped en resilience.Retry()
- [ ] Tool execution wrapped en circuit breaker
- [ ] Memory operations con retry
- [ ] Tests para cada path
```

### Paso 3: Health Checks (1-2 horas)
```
- [ ] Crear health checkers para LLM, Memory, Tools
- [ ] Registrar en startup
- [ ] Check en agent loop antes de operaciones críticas
```

### Paso 4: Metrics & Dashboards (1 hora)
```
- [ ] Llamar metrics.RecordErrorMetric() en error paths
- [ ] Llamar metrics.RecordRecovery() en recovery
- [ ] Verificar en dashboards
```

### Paso 5: Integration Tests (2-3 horas)
```
- [ ] End-to-end test: error → retry → recovery
- [ ] Test circuit breaker engagement
- [ ] Test fallback activation
```

**Total**: ~7-11 horas de desarrollo

---

## Monitoring Dashboard Integration

Una vez implementado Phase 4, los dashboards mostrarán:

### Dashboard 1: Error Rate
```
- LLM errors: rate(kairos_errors_total{error_code="llm_error"}[5m])
- Recovery rate: sum(rate(kairos_errors_recovered[5m])) / 
                 sum(rate(kairos_errors_total[5m]))
- Retry success: Tool calls recovered after retry
```

### Dashboard 2: Component Health
```
- LLM service health: kairos_health_status{component="llm"}
- Memory backend: kairos_health_status{component="memory"}
- Tool health: kairos_health_status{component="tools"}
```

### Dashboard 3: Resilience
```
- Circuit breaker state changes per component
- Fallback usage rate
- Timeout distribution
```

---

## Real-World Example: Tool Execution

Cómo se vería tool execution con error handling completo:

**Actual (sin error handling tipado)**:
```go
result, err := a.executeTool(ctx, toolCall)
if err != nil {
    slog.ErrorContext(ctx, "tool error", "error", err)
    continue // Try next iteration
}
```

**Con observabilidad completa (Fase 4)**:
```go
// CAPA 3: Retry + Circuit Breaker
toolCB := a.toolCircuitBreakers[toolCall.Name]
result, err := toolCB.Call(ctx, func() (interface{}, error) {
    return resilience.Retry(ctx, a.toolRetryConfig, func() (interface{}, error) {
        _, toolSpan := tracer.Start(ctx, "tool.execute",
            trace.WithAttributes(
                attribute.String("tool.name", toolCall.Name),
                attribute.String("tool.args", fmt.Sprint(toolCall.Args)),
            ),
        )
        defer toolSpan.End()
        
        return a.executeTool(ctx, toolCall)
    })
})

if err != nil {
    // CAPA 2: Clasificar error
    kairoErr := &errors.KairosError{
        Code: errors.CodeToolFailure,
        Msg: fmt.Sprintf("Tool %s failed", toolCall.Name),
        Recoverable: toolCall.Optional, // Tools opcionaels son recuperables
        Err: err,
    }
    
    // CAPA 1: Agregar contexto a span
    toolSpan.SetAttributes(
        attribute.String("error.code", kairoErr.Code.String()),
        attribute.Bool("error.recoverable", kairoErr.Recoverable),
    )
    
    // CAPA 3: Métrica + Recovery
    metrics.RecordErrorMetric(ctx, kairoErr, toolSpan)
    if kairoErr.Recoverable {
        metrics.RecordRecovery(ctx, kairoErr)
        slog.WarnContext(ctx, "tool failed but recovered",
            "tool", toolCall.Name,
            "reason", kairoErr.Msg,
        )
        continue
    }
    
    slog.ErrorContext(ctx, "tool failed permanently",
        "tool", toolCall.Name,
        "code", kairoErr.Code,
    )
    continue
}

// Success
a.appendToolResult(toolCall.Name, result)
```

---

## Testing Integration

Cómo testear la observabilidad integrada:

```go
func TestAgentLoopWithErrorRecovery(t *testing.T) {
    ctx := context.Background()
    
    // Setup agent con mock LLM que falla 2 veces
    mockLLM := &MockLLM{
        failCount: 2,
        failErr: errors.CodeTimeout,
    }
    agent := NewAgent(WithLLM(mockLLM))
    
    // Run agent
    err := agent.Run(ctx)
    require.NoError(t, err)
    
    // Verificar métricas
    metrics := agent.Metrics()
    
    // Debe haber 2 errores
    require.Equal(t, int64(2), metrics.ErrorsTotal.Sum())
    
    // Ambos deben recuperados (porque son timeouts, recuperables)
    require.Equal(t, int64(2), metrics.ErrorsRecovered.Sum())
    
    // Recovery rate = 100%
    rate := float64(metrics.ErrorsRecovered.Sum()) / 
            float64(metrics.ErrorsTotal.Sum())
    require.Equal(t, 1.0, rate)
}
```

---

## FAQ: Integration Questions

### P: ¿Se puede usar error handling sin cambiar el agent loop?

**R**: Sí, parcialmente. Puedes:
- Usar `KairosError` en funciones llamadas por el agent
- Usar retry/circuit breaker en funciones wrapper
- Pero no tendrás métricas automáticas sin cambiar el loop

Phase 4 hace una migración limpia que es optional (backward compatible).

### P: ¿Cuál es el overhead de estas capas?

**R**: Mínimo:
- **Trazas**: ~1% CPU overhead (ya está ahí)
- **Metrics**: ~0.5% (batched by OTEL)
- **Retry logic**: Solo cuando hay error
- **Health checks**: Cached (10s TTL por defecto)
- **Circuit breaker**: ~0.01ms por call (just state check)

Total: <2% overhead en happy path.

### P: ¿Puedo usar solo algunas capas?

**R**: Absolutamente. Son independientes:
- Trazas sin error handling ✓
- Error handling sin retry ✓
- Retry sin health checks ✓
- etc.

Combina lo que necesites.

### P: ¿Cómo debuggear si algo falla?

**R**: Las trazas tienen todo:
```
# En Datadog/Prometheus
trace_id: 123abc
span_id: abc456
  - error.code: llm_error
  - error.recoverable: true
  - error.attempt: 2
  - error.context: {"model": "gpt-4", "input_tokens": 1024}
```

Puedes reproducir exactamente qué sucedió.

---

## Next Steps

1. **Ahora** (Fase 3): Lee [OBSERVABILITY.md](OBSERVABILITY.md) para entender métricas
2. **Próximo** (Fase 4): Implementar pasos 1-5 del roadmap
3. **Testing**: Usar ejemplos en `examples/error-handling/` como guía
4. **Production**: Deploy con dashboards + alert rules

---

## Resources

- **[Error Handling Guide](ERROR_HANDLING.md)** - Typed errors & retry
- **[Observability Guide](OBSERVABILITY.md)** - Dashboards & alerts
- **[Examples](../examples/)** - Working code samples
- **[ADR 0005](internal/adr/0005-error-handling-strategy.md)** - Design decisions
- **[Agent Code](../pkg/agent/agent.go)** - Current implementation

---

**Version**: 1.0  
**Status**: Ready for Phase 4 implementation  
**Last Updated**: 2026-01-15

*Esta guía complementa ERROR_HANDLING.md con focus específico en agentes.*
