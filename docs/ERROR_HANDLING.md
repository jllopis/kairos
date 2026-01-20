# Manejo de Errores en Kairos

Kairos proporciona un sistema de manejo de errores tipado y observable que facilita la depuración, el monitoring y la recuperación automática.

## Inicio Rápido

```go
import (
    "github.com/jllopis/kairos/pkg/errors"
)

// Crear un error tipado
err := errors.New(errors.CodeToolFailure, "herramienta falló").
    WithContext("tool", "get_weather").
    WithRecoverable(true)

// Verificar tipo de error
if ke := errors.AsKairosError(err); ke != nil && ke.Code == errors.CodeToolFailure {
    // Manejar fallo de herramienta
}

// Obtener contexto
if ke := errors.AsKairosError(err); ke != nil {
    toolName := ke.Context["tool"]
    canRetry := ke.Recoverable
}
```

---

## Tipos de Error

Kairos define los siguientes códigos de error:

| Código | Descripción | Recuperable |
|--------|-------------|-------------|
| `CodeToolFailure` | Fallo en ejecución de herramienta | Depende del error |
| `CodeTimeout` | Timeout en operación | Sí |
| `CodeRateLimit` | Límite de tasa excedido | Sí |
| `CodeLLMError` | Error del proveedor LLM | Depende |
| `CodeMemoryError` | Error en operaciones de memoria | Depende |
| `CodeInternal` | Error interno del sistema | No |
| `CodeNotFound` | Recurso no encontrado | No |
| `CodeUnauthorized` | Sin autorización | No |
| `CodeInvalidInput` | Entrada inválida | No |

---

## Estructura KairosError

```go
type KairosError struct {
    Code        ErrorCode         // Código de error
    Message     string            // Mensaje legible
    Err         error             // Error original (opcional)
    Context     map[string]any    // Contexto adicional
    Attributes  map[string]string // Atributos OTEL
    Recoverable bool              // ¿Se puede reintentar?
}
```

### Métodos principales

```go
// Crear error
err := errors.New(errors.CodeLLMError, "modelo no disponible", nil)

// Añadir contexto
err = err.WithContext("model", "gpt-4").
         WithContext("provider", "openai")

// Marcar como recuperable
err = err.WithRecoverable(true)

// Wrap de error existente
err = errors.New(errors.CodeToolFailure, "ejecución falló", originalErr)

// Verificaciones
if ke := errors.AsKairosError(err); ke != nil && ke.Code == errors.CodeTimeout {
    // ¿Es timeout?
}
if ke := errors.AsKairosError(err); ke != nil && ke.Recoverable {
    // ¿Se puede reintentar?
}
```

---

## Patrones de Retry

Kairos incluye un sistema de retry con backoff exponencial:

```go
import "github.com/jllopis/kairos/pkg/resilience"

config := resilience.DefaultRetryConfig().
    WithMaxAttempts(3).
    WithInitialDelay(100 * time.Millisecond).
    WithMaxDelay(5 * time.Second)

err := config.Do(ctx, func() error {
    return callExternalAPI()
})
```

### Retry solo para errores recuperables

```go
import (
    "github.com/jllopis/kairos/pkg/errors"
    "github.com/jllopis/kairos/pkg/resilience"
)

config := resilience.DefaultRetryConfig()
var resp *llm.ChatResponse
err := config.Do(ctx, func() error {
    var err error
    resp, err = llm.Chat(ctx, messages)
    if err != nil {
        // Solo reintenta si es recuperable
        return errors.New(errors.CodeLLMError, "chat failed", err).
            WithRecoverable(isTransientError(err))
    }
    return nil
})
```

---

## Circuit Breaker

Protege contra fallos en cascada:

```go
import "github.com/jllopis/kairos/pkg/resilience"

cb := resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
    FailureThreshold: 5,           // Fallos antes de abrir
    SuccessThreshold: 2,           // Éxitos para cerrar en half-open
    Timeout:          30 * time.Second,
})

err := cb.Call(ctx, func() error {
    return callExternalService()
})

// Verificar estado
state := cb.State() // Closed, Open, HalfOpen
```

---

## Integración con Observabilidad

Los errores se integran automáticamente con OpenTelemetry:

```go
import (
    "github.com/jllopis/kairos/pkg/errors"
    "github.com/jllopis/kairos/pkg/telemetry"
)

// Registrar error en span actual
telemetry.RecordError(span, err)

// Registrar métricas de error
em, _ := telemetry.NewErrorMetrics(ctx)
em.RecordErrorMetric(ctx, err, "agent-llm")
em.RecordRecovery(ctx, errors.CodeTimeout)
```

### Métricas disponibles

| Métrica | Tipo | Descripción |
|---------|------|-------------|
| `kairos.errors.total` | Counter | Total de errores por código |
| `kairos.errors.recovered` | Counter | Errores recuperados exitosamente |
| `kairos.errors.rate` | Gauge | Tasa de errores por componente |
| `kairos.health.status` | Gauge | Estado de salud por componente |
| `kairos.circuitbreaker.state` | Gauge | Estado del circuit breaker |

### Atributos en trazas

Los errores añaden automáticamente estos atributos a los spans:

```
error.code = "TOOL_FAILURE"
error.recoverable = true
error.tool = "get_weather"
error.message = "timeout connecting to service"
```

---

## Uso en Agentes

El agent loop de Kairos usa errores tipados internamente:

```go
// Ejemplo: ejecutar herramienta con manejo de errores
func (a *Agent) executeTool(ctx context.Context, call llm.ToolCall) (any, error) {
    result, err := a.toolExecutor.Execute(ctx, call.Name, call.Args)
    if err != nil {
        // El error ya viene tipado desde el executor
        if ke := errors.AsKairosError(err); ke != nil && ke.Recoverable {
            // Intentar con herramienta alternativa si existe
            return a.tryFallback(ctx, call)
        }
        return nil, err
    }
    return result, nil
}
```

### Helpers del paquete agent

```go
import "github.com/jllopis/kairos/pkg/agent"

// Wrappers específicos para errores comunes
err := agent.WrapLLMError(originalErr, "model-name")
err := agent.WrapToolError(originalErr, "get_weather", "call-123")
err := agent.WrapMemoryError(originalErr, "store")
err := agent.WrapTimeoutError(originalErr, "agent-loop", maxIterations)
```

---

## Health Checks

Verificar salud de componentes antes de usarlos:

```go
import "github.com/jllopis/kairos/pkg/agent"

// Crear health checker para LLM
llmHealth := agent.NewLLMHealthChecker(llmProvider)
status := llmHealth.Check(ctx)

if status.Status != core.HealthStatusHealthy {
    // LLM no disponible, usar fallback o fallar gracefully
}

// Health checkers disponibles:
// - AgentHealthChecker
// - LLMHealthChecker  
// - MemoryHealthChecker
// - MCPHealthChecker
```

---

## Mapeo a gRPC (A2A)

En el servidor A2A, los errores se mapean a códigos gRPC:

| KairosError Code | gRPC Status |
|------------------|-------------|
| `CodeNotFound` | `NOT_FOUND` |
| `CodeInvalidInput` | `INVALID_ARGUMENT` |
| `CodeTimeout` | `DEADLINE_EXCEEDED` |
| `CodeUnauthorized` | `PERMISSION_DENIED` |
| `CodeRateLimit` | `RESOURCE_EXHAUSTED` |
| `CodeToolFailure` | `UNAVAILABLE` o `INTERNAL` |
| `CodeLLMError` | `UNAVAILABLE` |
| `CodeInternal` | `INTERNAL` |

---

## Ejemplo Completo

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/jllopis/kairos/pkg/errors"
    "github.com/jllopis/kairos/pkg/resilience"
)

func main() {
    ctx := context.Background()
    
    // Configurar retry
    retryConfig := resilience.DefaultRetryConfig().
        WithMaxAttempts(3).
        WithInitialDelay(500 * time.Millisecond).
        WithMaxDelay(10 * time.Second)

    var result string
    err := retryConfig.Do(ctx, func() error {
        // Tu código aquí
        resp, err := callExternalAPI()
        if err != nil {
            return errors.New(errors.CodeToolFailure, "API call failed", err).
                WithContext("endpoint", "/api/data").
                WithRecoverable(isNetworkError(err))
        }
        result = resp
        return nil
    })
    
    if err != nil {
        if ke := errors.AsKairosError(err); ke != nil {
            log.Printf("Error [%s]: %s (recuperable: %v)", 
                ke.Code, ke.Message, ke.Recoverable)
        }
        return
    }
    
    fmt.Println("Resultado:", result)
}
```

---

## Documentación Relacionada

- **[Guía de Integración](INTEGRATION_GUIDE.md)**: Uso del manejo de errores en loops de agentes
- **[Guía de Observabilidad](OBSERVABILITY.md)**: Dashboards y alertas
- **[Exportación de Métricas](METRICS_EXPORT.md)**: Configuración OTLP
