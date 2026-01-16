# 10 - Resilience Patterns

Patrones de resiliencia: retry, circuit breaker, timeouts y fallbacks.

## Qué aprenderás

- Retry con backoff exponencial
- Circuit breaker para proteger contra cascadas
- Timeouts por operación
- Estrategias de fallback
- Health checks de componentes

## Ejecutar

```bash
go run .
```

## Retry con backoff

```go
import "github.com/jllopis/kairos/pkg/errors"

config := errors.RetryConfig{
    MaxAttempts:  3,
    InitialDelay: 100 * time.Millisecond,
    MaxDelay:     5 * time.Second,
    Multiplier:   2.0,  // Backoff exponencial
}

result, err := errors.Retry(ctx, config, func() (string, error) {
    return callExternalAPI()
})
```

## Circuit Breaker

```go
cb := errors.NewCircuitBreaker(errors.CircuitBreakerConfig{
    FailureThreshold: 5,           // Fallos antes de abrir
    ResetTimeout:     30 * time.Second,
    HalfOpenRequests: 2,           // Pruebas en half-open
})

result, err := cb.Execute(ctx, func() (string, error) {
    return callUnreliableService()
})

// Estados: Closed → Open → HalfOpen → Closed
```

## Timeouts

```go
// Timeout por operación
ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
defer cancel()

result, err := ag.Run(ctx, "consulta larga")
if errors.IsCode(err, errors.CodeTimeout) {
    // Manejar timeout
}
```

## Fallback strategies

```go
// Fallback estático
result, err := withFallback(ctx, primaryFunc, "valor por defecto")

// Fallback con caché
result, err := withCachedFallback(ctx, primaryFunc, cache)

// Fallback encadenado
result, err := withChainedFallback(ctx, 
    primaryProvider,
    backupProvider,
    emergencyProvider,
)
```

## Health Checks

```go
import "github.com/jllopis/kairos/pkg/agent"

// Verificar salud del LLM
health := agent.NewLLMHealthChecker(llmProvider)
status := health.Check(ctx)

if status.Status != core.HealthStatusHealthy {
    // Activar modo degradado
}
```

## Siguiente paso

→ [11-observability](../11-observability/) para métricas y dashboards
