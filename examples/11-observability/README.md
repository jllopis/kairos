# 11 - Observability

Métricas, trazas y alertas con OpenTelemetry.

## Qué aprenderás

- Configurar exportadores OTEL (stdout, OTLP)
- Métricas de errores y recuperación
- Trazas distribuidas entre componentes
- Configurar alertas basadas en métricas

## Ejecutar

```bash
go run .
```

## Configuración OTEL

```go
import "github.com/jllopis/kairos/pkg/telemetry"

cfg := telemetry.Config{
    ServiceName: "my-agent",
    Exporter:    "otlp",  // o "stdout" para desarrollo
    Endpoint:    "localhost:4317",
}

shutdown, _ := telemetry.Init(ctx, cfg)
defer shutdown(ctx)
```

## Métricas disponibles

| Métrica | Tipo | Descripción |
|---------|------|-------------|
| `kairos.errors.total` | Counter | Errores por código |
| `kairos.errors.recovered` | Counter | Recuperaciones exitosas |
| `kairos.retry.attempts` | Histogram | Intentos de retry |
| `kairos.circuitbreaker.state` | Gauge | Estado del CB |
| `kairos.health.status` | Gauge | Salud de componentes |
| `kairos.llm.latency_ms` | Histogram | Latencia LLM |
| `kairos.tool.duration_ms` | Histogram | Duración de tools |

## Registrar métricas

```go
import "github.com/jllopis/kairos/pkg/telemetry"

// Error con métrica automática
telemetry.RecordErrorMetric(ctx, err, span)

// Recuperación exitosa
telemetry.RecordRecovery(ctx, err)
```

## Trazas

```go
import "go.opentelemetry.io/otel"

tracer := otel.Tracer("kairos.agent")
ctx, span := tracer.Start(ctx, "agent.run")
defer span.End()

// Atributos en el span
span.SetAttributes(
    attribute.String("agent.id", "my-agent"),
    attribute.Int("iteration", 1),
)

// Registrar error en span
span.RecordError(err)
```

## Queries útiles (Prometheus/Grafana)

```promql
# Tasa de errores por minuto
rate(kairos_errors_total[1m])

# Recovery rate
kairos_errors_recovered / kairos_errors_total

# P95 latencia LLM
histogram_quantile(0.95, kairos_llm_latency_ms_bucket)
```

## Siguiente paso

→ [12-production-layout](../12-production-layout/) para estructura enterprise
