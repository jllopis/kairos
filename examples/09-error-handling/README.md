# 09 - Error Handling

Sistema de errores tipados con contexto rico y observabilidad.

## Qué aprenderás

- Crear errores tipados con `KairosError`
- Distinguir errores recuperables vs. fatales
- Añadir contexto para debugging
- Integración automática con OpenTelemetry

## Ejecutar

```bash
go run .
```

## Tipos de error

| Código | Descripción | Recuperable |
|--------|-------------|-------------|
| `CodeToolFailure` | Fallo en herramienta | Depende |
| `CodeTimeout` | Timeout | Sí |
| `CodeRateLimit` | Rate limit | Sí |
| `CodeLLMError` | Error del LLM | Depende |
| `CodeMemoryError` | Error de memoria | Depende |
| `CodeNotFound` | No encontrado | No |
| `CodeUnauthorized` | Sin autorización | No |

## Código clave

```go
import "github.com/jllopis/kairos/pkg/errors"

// Crear error tipado
err := errors.New(errors.CodeToolFailure, "API no disponible").
    WithContext("tool", "get_weather").
    WithContext("endpoint", "api.weather.com").
    WithRecoverable(true)

// Verificar tipo
if errors.IsCode(err, errors.CodeToolFailure) {
    // Manejar fallo de herramienta
}

// Verificar si se puede reintentar
if errors.IsRecoverable(err) {
    // Retry logic
}

// Obtener contexto completo
if ke, ok := errors.AsKairosError(err); ok {
    tool := ke.Context["tool"]
    fmt.Printf("Tool %s falló: %s\n", tool, ke.Message)
}
```

## Integración OTEL

Los errores se registran automáticamente en spans:

```go
// En el span actual
span.SetAttributes(
    attribute.String("error.code", "TOOL_FAILURE"),
    attribute.String("error.tool", "get_weather"),
    attribute.Bool("error.recoverable", true),
)
```

## Siguiente paso

→ [10-resilience-patterns](../10-resilience-patterns/) para retry y circuit breaker
