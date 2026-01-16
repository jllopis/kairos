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
if errors.IsCode(err, errors.CodeToolFailure) {
    // Manejar fallo de herramienta
}

// Obtener contexto
if ke, ok := errors.AsKairosError(err); ok {
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
err := errors.New(errors.CodeLLMError, "modelo no disponible")

// Añadir contexto
err = err.WithContext("model", "gpt-4").
         WithContext("provider", "openai")

// Marcar como recuperable
err = err.WithRecoverable(true)

// Wrap de error existente
err = errors.Wrap(originalErr, errors.CodeToolFailure, "ejecución falló")

// Verificaciones
errors.IsCode(err, errors.CodeTimeout)      // ¿Es timeout?
errors.IsRecoverable(err)                   // ¿Se puede reintentar?
ke, ok := errors.AsKairosError(err)         // Obtener KairosError
```

---

## Patrones de Retry

Kairos incluye un sistema de retry con backoff exponencial:

```go
import "github.com/jllopis/kairos/pkg/errors"

config := errors.RetryConfig{
    MaxAttempts:  3,
    InitialDelay: 100 * time.Millisecond,
    MaxDelay:     5 * time.Second,
    Multiplier:   2.0,
}

result, err := errors.Retry(ctx, config, func() (string, error) {
    return callExternalAPI()
})
```

### Retry solo para errores recuperables

```go
result, err := errors.RetryRecoverable(ctx, config, func() (string, error) {
    resp, err := llm.Chat(ctx, messages)
    if err != nil {
        // Solo reintenta si es recuperable
        return "", errors.Wrap(err, errors.CodeLLMError, "chat failed").
            WithRecoverable(isTransientError(err))
    }
    return resp, nil
})
```

---

## Circuit Breaker

Protege contra fallos en cascada:

```go
import "github.com/jllopis/kairos/pkg/errors"

cb := errors.NewCircuitBreaker(errors.CircuitBreakerConfig{
    FailureThreshold: 5,           // Fallos antes de abrir
    ResetTimeout:     30 * time.Second,
    HalfOpenRequests: 2,           // Requests en half-open
})

result, err := cb.Execute(ctx, func() (string, error) {
    return callExternalService()
})

// Verificar estado
state := cb.State() // Closed, Open, HalfOpen
```

---

## Integración con Observabilidad

Los errores se integran automáticamente con OpenTelemetry:

```go
import "github.com/jllopis/kairos/pkg/telemetry"

// Registrar error en span actual
telemetry.RecordErrorMetric(ctx, err, span)

// Registrar recuperación exitosa
telemetry.RecordRecovery(ctx, err)
```

### Métricas disponibles

| Métrica | Tipo | Descripción |
|---------|------|-------------|
| `kairos.errors.total` | Counter | Total de errores por código |
| `kairos.errors.recovered` | Counter | Errores recuperados exitosamente |
| `kairos.retry.attempts` | Histogram | Intentos de retry |
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
        if errors.IsRecoverable(err) {
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
err := agent.WrapLLMError(originalErr, "chat completion failed")
err := agent.WrapToolError(originalErr, "get_weather", args)
err := agent.WrapMemoryError(originalErr, "store", key)
err := agent.WrapTimeoutError(operation, duration)
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
    "time"
    
    "github.com/jllopis/kairos/pkg/agent"
    "github.com/jllopis/kairos/pkg/errors"
)

func main() {
    ctx := context.Background()
    
    // Configurar retry
    retryConfig := errors.RetryConfig{
        MaxAttempts:  3,
        InitialDelay: 500 * time.Millisecond,
        MaxDelay:     10 * time.Second,
        Multiplier:   2.0,
    }
    
    // Ejecutar con retry automático
    result, err := errors.RetryRecoverable(ctx, retryConfig, func() (string, error) {
        // Tu código aquí
        resp, err := callExternalAPI()
        if err != nil {
            return "", errors.Wrap(err, errors.CodeToolFailure, "API call failed").
                WithContext("endpoint", "/api/data").
                WithRecoverable(isNetworkError(err))
        }
        return resp, nil
    })
    
    if err != nil {
        if ke, ok := errors.AsKairosError(err); ok {
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
