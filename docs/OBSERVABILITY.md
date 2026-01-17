# Guía de Observabilidad

Kairos integra **OpenTelemetry (OTEL)** desde el inicio, proporcionando visibilidad completa en:

- **Errores**: Clasificación por tipo, componente, recoverabilidad
- **Resiliencia**: Reintentos, circuit breaker, fallbacks, timeouts
- **Salud**: Estado de componentes, degradación, recuperación
- **Rendimiento**: Tasa de errores, latencia de recuperación

## Table of Contents

1. [Introducción](#introduccion)
2. [Arquitectura de Observabilidad](#arquitectura-de-observabilidad)
3. [Atributos de Span Enriquecidos](#atributos-de-span-enriquecidos)
4. [Métricas Disponibles](#metricas-disponibles)
5. [Dashboards](#dashboards)
6. [Reglas de Alerta](#reglas-de-alerta)
7. [Ejemplos de Uso](#ejemplos-de-uso)
8. [Integración con Backends](#integracion-con-backends)
9. [SLOs y Recomendaciones](#slos-y-recomendaciones)

---

## Introducción

Kairos integra **OpenTelemetry (OTEL)** desde el inicio, proporcionando visibilidad completa en:

- **Errores**: Clasificación por tipo, componente, recoverabilidad
- **Resiliencia**: Reintentos, circuit breaker, fallbacks, timeouts
- **Salud**: Estado de componentes, degradación, recuperación
- **Rendimiento**: Tasa de errores, latencia de recuperación

### ¿Por qué esto importa?

Sin observabilidad, no sabes si tus patrones de resiliencia funcionan. Con Kairos:

- ✅ **Visibilidad Real**: Cada error se registra con contexto completo
- ✅ **Detección Temprana**: Alertas antes de que se propague el problema
- ✅ **Debugging Rápido**: Dashboards con correlaciones error ↔ recuperación
- ✅ **SLOs Medibles**: Datos para definir y cumplir SLOs de confiabilidad

---

## Arquitectura de Observabilidad

```
┌─────────────────────────────────────────────────────────────┐
│                    Aplicación Kairos                        │
├─────────────────────────────────────────────────────────────┤
│  pkg/errors → KairosError (typed errors)                   │
│  pkg/resilience → Retry, CircuitBreaker, Fallback, etc.    │
│  pkg/telemetry → RecordError(), RecordErrorMetric()        │
├─────────────────────────────────────────────────────────────┤
│  OTEL SDK: Metrics + Traces                                 │
├─────────────────────────────────────────────────────────────┤
│                    OTEL Exporters                           │
│  ├─ Stdout (desarrollo)                                     │
│  ├─ gRPC/OTLP (producción: Datadog, New Relic, etc.)       │
│  └─ Jaeger, Prometheus, etc.                               │
├─────────────────────────────────────────────────────────────┤
│              Backends & Visualización                       │
│  ├─ Prometheus (métricas + alertas)                        │
│  ├─ Grafana (dashboards)                                    │
│  └─ AlertManager (notificaciones)                           │
└─────────────────────────────────────────────────────────────┘
```

### Flujo de Datos

```
Error ocurre
    ↓
KairosError creado con código + contexto
    ↓
telemetry.RecordError(span, err) → Atributos en trace
    ↓
metrics.RecordErrorMetric(ctx, err, "component") → Contador OTEL
    ↓
OTEL Exporter envía a backend (OTLP, Prometheus, etc.)
    ↓
Dashboards visualizan → Alertas disparan si es necesario
    ↓
Equipo responde basado en datos
```

---

## Atributos de Span Enriquecidos

Kairos añade atributos semánticos ricos a los spans de OTEL para facilitar debugging y análisis.

### Spans Disponibles

| Span | Descripción |
|------|-------------|
| `Agent.Run` | Ejecución completa del agente |
| `Agent.LLM.Chat` | Llamada al LLM |
| `Agent.Tool.Call` | Ejecución de una tool |

### Atributos del Agente (`Agent.Run`)

| Atributo | Tipo | Descripción |
|----------|------|-------------|
| `kairos.agent.id` | string | Identificador del agente |
| `kairos.agent.role` | string | Rol del agente |
| `kairos.agent.model` | string | Modelo LLM usado |
| `kairos.agent.run_id` | string | ID único de la ejecución |
| `kairos.agent.max_iterations` | int | Límite de iteraciones |
| `kairos.session.id` | string | ID de sesión (si hay conversación) |
| `kairos.conversation.enabled` | bool | Si hay memoria de conversación |
| `kairos.conversation.message_count` | int | Mensajes en historial |
| `kairos.tools.count` | int | Total de tools disponibles |
| `kairos.tools.local_count` | int | Tools locales |
| `kairos.tools.mcp_count` | int | Tools MCP |
| `kairos.tools.skill_count` | int | Skills como tools |
| `kairos.tools.names` | []string | Nombres de tools |
| `kairos.memory.enabled` | bool | Si hay memoria semántica |
| `kairos.memory.type` | string | Tipo de memoria |
| `kairos.task.id` | string | ID de task (si existe) |
| `kairos.task.goal` | string | Objetivo de la task |
| `kairos.task.status` | string | Estado de la task |

### Atributos del LLM (`Agent.LLM.Chat`)

| Atributo | Tipo | Descripción |
|----------|------|-------------|
| `gen_ai.request.model` | string | Modelo solicitado |
| `gen_ai.system` | string | Provider (openai, anthropic...) |
| `gen_ai.request.messages` | int | Número de mensajes enviados |
| `gen_ai.tool_calls` | int | Tool calls en la respuesta |
| `gen_ai.usage.input_tokens` | int | Tokens de entrada |
| `gen_ai.usage.output_tokens` | int | Tokens de salida |
| `gen_ai.duration_ms` | float | Duración en ms |
| `gen_ai.finish_reason` | string | Razón de finalización |

### Atributos de Tool (`Agent.Tool.Call`)

| Atributo | Tipo | Descripción |
|----------|------|-------------|
| `kairos.tool.name` | string | Nombre de la tool |
| `kairos.tool.call_id` | string | ID de la llamada |
| `kairos.tool.source` | string | Origen: "local", "mcp", "skill" |
| `kairos.tool.duration_ms` | float | Duración en ms |
| `kairos.tool.success` | bool | Si tuvo éxito |
| `kairos.tool.arguments` | string | Argumentos (truncados) |
| `kairos.tool.result` | string | Resultado (truncado) |

### Ejemplo en Jaeger

```
Agent.Run (350ms)
├── kairos.agent.id: "assistant"
├── kairos.agent.model: "gpt-4"
├── kairos.tools.count: 3
├── kairos.conversation.enabled: true
├── kairos.session.id: "user-123"
│
├── Agent.LLM.Chat (200ms)
│   ├── gen_ai.request.model: "gpt-4"
│   ├── gen_ai.request.messages: 5
│   └── gen_ai.tool_calls: 1
│
└── Agent.Tool.Call (100ms)
    ├── kairos.tool.name: "search"
    ├── kairos.tool.source: "mcp"
    ├── kairos.tool.success: true
    └── kairos.tool.duration_ms: 98.5
```

---

## Métricas Disponibles

Kairos expone **5 métricas principales** vía OTEL:

### 1. `kairos.errors.total` (Counter)

**Descripción**: Número total de errores por código de error y componente.

**Atributos**:
- `error.code`: Código del error (TOOL_FAILURE, TIMEOUT, LLM_ERROR, etc.)
- `component`: Nombre del componente que reporta el error
- `recoverable`: "true" o "false"

**Ejemplo**:
```
kairos.errors.total{error.code="TIMEOUT", component="llm-service", recoverable="true"} = 42
kairos.errors.total{error.code="TOOL_FAILURE", component="executor", recoverable="false"} = 3
```

**Uso**: Medir tasa de errores, identificar componentes problemáticos, alertar sobre no-recuperables.

---

### 2. `kairos.errors.recovered` (Counter)

**Descripción**: Número de errores recuperados exitosamente (reintentos que funcionaron, fallbacks usados).

**Atributos**:
- `error.code`: Código del error recuperado

**Ejemplo**:
```
kairos.errors.recovered{error.code="TIMEOUT"} = 38  (de 42 totales)
kairos.errors.recovered{error.code="TOOL_FAILURE"} = 0  (de 3 no-recuperables)
```

**Uso**: Calcular **tasa de recuperación** = `errors.recovered / errors.total` (objetivo: >80%)

---

### 3. `kairos.errors.rate` (Gauge)

**Descripción**: Tasa de errores por minuto por componente.

**Atributos**:
- `component`: Nombre del componente

**Ejemplo**:
```
kairos.errors.rate{component="llm-service"} = 2.5    (2.5 errores/min)
kairos.errors.rate{component="executor"} = 0.1       (0.1 errores/min)
```

**Uso**: Umbral para alertas. Valores de referencia:
- Normal: < 1 error/min
- Advertencia: 1-5 errores/min
- Crítico: > 5 errores/min

---

### 4. `kairos.health.status` (Gauge)

**Descripción**: Estado de salud del componente en el momento de la medición.

**Valores**:
- `2` = HEALTHY (operativo normalmente)
- `1` = DEGRADED (funciona pero con limitaciones)
- `0` = UNHEALTHY (no funciona, usando fallback)

**Atributos**:
- `component`: Nombre del componente

**Ejemplo**:
```
kairos.health.status{component="llm-service"} = 2      (verde)
kairos.health.status{component="cache"} = 1            (amarillo)
kairos.health.status{component="external-api"} = 0     (rojo)
```

**Uso**: 
- Dashboards: Grid de colores (rojo/amarillo/verde)
- Routing: Desviar tráfico de componentes no-saludables
- Alertas: UNHEALTHY → investigación inmediata

---

### 5. `kairos.circuitbreaker.state` (Gauge)

**Descripción**: Estado actual del circuit breaker para cada componente.

**Valores**:
- `2` = CLOSED (operando normalmente, solicitudes fluyendo)
- `1` = HALF_OPEN (probando recuperación, solicitudes limitadas)
- `0` = OPEN (circuito roto, fallback activo, solicitudes rechazadas)

**Atributos**:
- `component`: Nombre del componente

**Ejemplo**:
```
kairos.circuitbreaker.state{component="api-client"} = 2         (cerrado)
kairos.circuitbreaker.state{component="external-service"} = 1   (medio-abierto)
kairos.circuitbreaker.state{component="failing-dep"} = 0        (abierto)
```

**Uso**:
- Entender cascadas de fallos
- Identificar qué servicios están afectando a otros
- Medir tiempo de recuperación

---

## Dashboards

### Dashboard 1: Error Rate & Recovery (Tasa de Errores y Recuperación)

**Propósito**: Entender la salud general del sistema en términos de errores y resiliencia.

#### Panel 1.1: Error Rate por Código (Últimas 24h)

**Query (PromQL)**:
```promql
rate(kairos.errors.total{error_code=~".+"}[5m])
```

**Configuración Grafana**:
- **Tipo**: Line Chart
- **Eje X**: Tiempo (5m bucket)
- **Eje Y**: Errores por segundo
- **Leyenda**: Por `error_code` (TOOL_FAILURE, TIMEOUT, LLM_ERROR, etc.)
- **Colores**: Rojo para críticos (CodeInternal, CodeMemoryError), naranja para recuperables

**Interpretación**:
- Línea suave y baja: Sistema sano ✅
- Picos ocasionales: Dentro de lo normal si se recupera
- Línea constantemente alta: **Alerta** → investigar causa raíz

---

#### Panel 1.2: Tasa de Recuperación (%)

**Query (PromQL)**:
```promql
(
  rate(kairos.errors.recovered[5m]) 
  / 
  rate(kairos.errors.total[5m])
) * 100
```

**Configuración Grafana**:
- **Tipo**: Gauge
- **Umbral Inferior**: 80% (amarillo)
- **Umbral Superior**: 90% (verde)
- **Unidad**: percent

**Interpretación**:
- Verde (>90%): Excelente resiliencia ✅
- Amarillo (80-90%): Aceptable, monitorear
- Rojo (<80%): **Problema** → revisar configuración de reintentos/fallbacks

---

#### Panel 1.3: Error Rate por Componente (Gauge)

**Query (PromQL)**:
```promql
sum(rate(kairos.errors.total[5m])) by (component)
```

**Configuración Grafana**:
- **Tipo**: Table (mostrar top 5-10)
- **Columnas**: component, error_rate
- **Ordenar por**: error_rate DESC
- **Colores**: 
  - Rojo: > 5 errores/min
  - Naranja: 1-5 errores/min
  - Verde: < 1 error/min

**Interpretación**:
- Identifica qué componentes generan más errores
- Ejemplo: si `llm-service` está en rojo → investigar calidad del modelo o sobrecarga

---

### Dashboard 2: Component Health (Salud de Componentes)

**Propósito**: Monitoreo en tiempo real del estado operativo de cada componente.

#### Panel 2.1: Health Status Grid

**Query (PromQL)**:
```promql
kairos.health.status{component=~".+"}
```

**Configuración Grafana**:
- **Tipo**: Stat or Status Grid
- **Mostrar**: Valor actual (0, 1, 2)
- **Color Mapping**:
  - 0 → Rojo (UNHEALTHY)
  - 1 → Amarillo (DEGRADED)
  - 2 → Verde (HEALTHY)
- **Layout**: Grid (4-5 columnas)

**Componentes típicos a monitorear**:
- llm-service
- cache
- database
- external-api
- tool-executor
- memory

**Interpretación**:
- **Verde**: ✅ Todo está bien
- **Amarillo**: ⚠️ Observar, puede empeorar
- **Rojo**: 🔴 ACCIÓN REQUERIDA → fallback activo, investigar

---

#### Panel 2.2: Circuit Breaker States

**Query (PromQL)**:
```promql
kairos.circuitbreaker.state{component=~".+"}
```

**Configuración Grafana**:
- **Tipo**: Status Panels (uno por componente importante)
- **Color Mapping**:
  - 2 → Verde (CLOSED: operando normalmente)
  - 1 → Naranja (HALF_OPEN: probando recuperación)
  - 0 → Rojo (OPEN: fallback activo)
- **Mostrar**: Estado actual + última actualización

**Interpretación**:
- **CLOSED**: Flujo normal ✅
- **HALF_OPEN**: En recuperación, monitorear próximos minutos
- **OPEN**: Problema crítico, usar fallback, investigar dependencia

---

#### Panel 2.3: Health Timeline (Últimas 24h)

**Query (PromQL)**:
```promql
changes(kairos.health.status{component=~".+"}[1h])
```

**Configuración Grafana**:
- **Tipo**: Time Series Heatmap
- **Eje X**: Tiempo (1h bucket)
- **Eje Y**: Componente
- **Colores**: Intensidad = frecuencia de cambios

**Interpretación**:
- Líneas suave: Componente estable ✅
- Muchas líneas: Componente inestable ⚠️
- Largo período rojo: Outage 🔴

---

### Dashboard 3: Error Details (Detalles de Errores)

**Propósito**: Deep dive en patrones específicos de errores.

#### Panel 3.1: Error Breakdown Table

**Query (PromQL)**:
```promql
sum(rate(kairos.errors.total[5m])) by (error_code, component, recoverable)
```

**Configuración Grafana**:
- **Tipo**: Table
- **Columnas**: error_code, component, recoverable, rate
- **Ordenar por**: rate DESC
- **Filtros**: Permitir drill-down por error_code

**Interpretación**:
- ¿Qué error es más frecuente? TIMEOUT > TOOL_FAILURE > LLM_ERROR
- ¿Cuál es menos recuperable? Buscar non-recoverable
- Ejemplo:
  ```
  TOOL_FAILURE | executor | false | 0.05 errors/sec ← CRÍTICO
  TIMEOUT      | llm-svc  | true  | 2.50 errors/sec ← Normal
  ```

---

#### Panel 3.2: Timeout vs Circuit Breaker Correlation

**Query (PromQL)**:
```promql
# Gráfico dual axis
Eje 1: rate(kairos.errors.total{error_code="TIMEOUT"}[5m])
Eje 2: kairos.circuitbreaker.state{component=~".+"}
```

**Configuración Grafana**:
- **Tipo**: Time Series (dual axis)
- **Eje Izquierdo**: Error rate (línea roja)
- **Eje Derecho**: Circuit state (línea azul escalonada)

**Interpretación**:
- ¿Los timeouts causan que circuit breaker abra?
- Ejemplo: Línea roja sube → línea azul pasa de 2→0 (CLOSED→OPEN)
- Esto sugiere que timeouts disparan el circuit breaker (comportamiento esperado)

---

#### Panel 3.3: Recovery Latency

**Query (PromQL)**:
```promql
# Medida: tiempo entre error y recuperación exitosa
# (Requiere instrumentación manual: timestamp error + timestamp recovery)
histogram_quantile(0.95, rate(kairos.errors.recovered[5m]))
```

**Configuración Grafana**:
- **Tipo**: Stat (p95 latencia)
- **Unidad**: ms o s
- **Umbral**: < 5s ideal

**Interpretación**:
- p95 < 2s: Excelente resiliencia ✅
- p95 > 10s: Lento, revisar configuración de reintentos

---

## Reglas de Alerta

### Arquitectura de Alertas

```
Prometheus (evalúa reglas cada 30s)
    ↓
Regla dispara si condición es verdadera
    ↓
AlertManager recibe alert
    ↓
Enruta a: Slack, PagerDuty, Email, etc.
    ↓
Equipo actúa
```

### Alert 1: High Error Rate (Tasa de Errores Alta)

**Nombre**: `KairosHighErrorRate`

**Regla (Prometheus)**:
```yaml
alert: KairosHighErrorRate
expr: rate(kairos.errors.total[5m]) > 10
for: 2m
severity: critical
annotations:
  summary: "Kairos error rate muy alta"
  description: |
    Tasa de errores: {{ $value }} errors/sec (umbral: 10)
    Componente: {{ $labels.component }}
    Código: {{ $labels.error_code }}
  action: "Verificar logs del componente, posible sobrecarga o dependencia caída"
```

**Cuándo dispara**:
- Más de 10 errores/segundo durante 2+ minutos consecutivos

**Acciones**:
1. ✅ Verificar logs: `kubectl logs -f <pod>`
2. ✅ Revisar estado de dependencias: ¿LLM caído? ¿BD lenta?
3. ✅ Aumentar recursos si es sobrecarga
4. ✅ Rollback si fue cambio reciente

**Remediación**:
```bash
# Escalar servicio
kubectl scale deployment llm-service --replicas=3

# Ver último error
kubectl logs <pod> --tail=100 | grep ERROR

# Reiniciar si es necesario
kubectl rollout restart deployment llm-service
```

---

### Alert 2: Low Recovery Rate (Tasa de Recuperación Baja)

**Nombre**: `KairosLowRecoveryRate`

**Regla (Prometheus)**:
```yaml
alert: KairosLowRecoveryRate
expr: |
  (
    rate(kairos.errors.recovered[5m]) 
    / 
    rate(kairos.errors.total[5m])
  ) < 0.8
for: 5m
severity: warning
annotations:
  summary: "Tasa de recuperación de errores baja"
  description: |
    Recovery rate: {{ $value | humanizePercentage }} (meta: >= 80%)
    Recuperados: {{ $value_recovered }}
    Totales: {{ $value_total }}
  action: "Revisar configuración de reintentos y fallbacks"
```

**Cuándo dispara**:
- Menos del 80% de errores se recuperan durante 5+ minutos

**Acciones**:
1. ✅ Revisar qué errores no se recuperan: `kairos.errors.recovered / kairos.errors.total by error_code`
2. ✅ ¿Son non-recoverable por diseño? (p.ej., CodeUnauthorized)
3. ✅ ¿Fallan los reintentos? Aumentar MaxAttempts
4. ✅ ¿Fallback no configurado? Activar fallback strategy

**Remediación**:
```go
// Aumentar intentos de reintento
retryConfig := resilience.DefaultRetryConfig().
    WithMaxAttempts(5).  // Era 3
    WithInitialDelay(100 * time.Millisecond)

// Activar fallback
fallback := &resilience.CachedFallback{
    Cache: lastKnownGoodValue,
}
```

---

### Alert 3: Circuit Breaker Open (Circuito Abierto)

**Nombre**: `KairosCircuitBreakerOpen`

**Regla (Prometheus)**:
```yaml
alert: KairosCircuitBreakerOpen
expr: kairos.circuitbreaker.state{component=~".+"} == 0
for: 1m
severity: critical
annotations:
  summary: "Circuit breaker abierto"
  description: |
    Componente: {{ $labels.component }}
    Estado: OPEN (usando fallback)
    Fallback puede ser más lento o degradado
  action: |
    1. Investigar por qué el servicio falla
    2. Verificar dependencias (DB, APIs externas)
    3. Una vez estable, circuit breaker se auto-resetea
```

**Cuándo dispara**:
- Circuit breaker abre (demasiados errores consecutivos) por 1+ minuto

**Acciones**:
1. ✅ Identificar causaraíz: ¿Por qué falla el servicio?
2. ✅ Ver estado de dependencias
3. ✅ Reiniciar servicio si es necesario
4. ✅ Escalar recursos si es sobrecarga

**Remediación**:
```bash
# Ver estado de dependencia
curl https://external-api.com/health

# Si la dependencia está down, contactar al equipo
# El circuit breaker se auto-recuperará cuando la dependencia se recupere

# En ~30s en HALF_OPEN, probará una solicitud
# Si tiene éxito, pasará a CLOSED automáticamente
```

---

### Alert 4: Component Degraded (Componente Degradado)

**Nombre**: `KairosComponentDegraded`

**Regla (Prometheus)**:
```yaml
alert: KairosComponentDegraded
expr: kairos.health.status{component=~".+"} == 1
for: 3m
severity: warning
annotations:
  summary: "Componente degradado"
  description: |
    Componente: {{ $labels.component }}
    Estado: DEGRADED (funciona pero con limitaciones)
    Puede empeorar → prepararse para fallback
  action: "Monitorear, estar listo para escalar o conmutar fallback"
```

**Cuándo dispara**:
- Componente en estado DEGRADED por 3+ minutos

**Acciones**:
1. ✅ Monitorear tasa de errores → ¿mejorando o empeorando?
2. ✅ Si empeora, conmutar a fallback manualmente
3. ✅ Si mejora, esperar a que se recupere completamente
4. ✅ Post-mortem: ¿por qué degradó?

---

### Alert 5: Component Unhealthy (Componente No Saludable)

**Nombre**: `KairosComponentUnhealthy`

**Regla (Prometheus)**:
```yaml
alert: KairosComponentUnhealthy
expr: kairos.health.status{component=~".+"} == 0
for: 1m
severity: critical
annotations:
  summary: "⚠️ COMPONENTE NO SALUDABLE - ACCIÓN INMEDIATA"
  description: |
    Componente: {{ $labels.component }}
    Estado: UNHEALTHY (usando fallback, posible outage)
    
    Datos de tráfico:
    - Error rate: {{ $value_error_rate }}
    - Circuit breaker: OPEN
    - Fallback activo
  action: |
    🚨 INVESTIGAR YA:
    1. ¿Está la dependencia down?
    2. ¿Se agotó capacidad?
    3. ¿Problema de red?
    → Reiniciar, escalar, o conmutar a secondary
```

**Cuándo dispara**:
- Componente en estado UNHEALTHY por 1+ minuto

**Acciones** (CRÍTICAS):
1. 🚨 Investigar causa raíz inmediatamente
2. 🚨 Escalar infraestructura si es necesario
3. 🚨 Conmutar a servicio secondary si está disponible
4. 🚨 Comunicar a stakeholders si es outage público

**Remediación**:
```bash
# 1. Ver logs
kubectl logs <pod> --tail=200 | grep -i error

# 2. Ver recursos
kubectl describe pod <pod>

# 3. Escalar si es necesario
kubectl scale deployment llm-service --replicas=5

# 4. Si problem persiste, rollback cambio reciente
git revert <commit>

# 5. Conmutar fallback (si está configurado)
# Esto es manejo manual de la aplicación
```

---

### Alert 6: Non-Recoverable Errors (Errores No Recuperables)

**Nombre**: `KairosNonRecoverableErrors`

**Regla (Prometheus)**:
```yaml
alert: KairosNonRecoverableErrors
expr: rate(kairos.errors.total{recoverable="false"}[5m]) > 1
for: 2m
severity: critical
annotations:
  summary: "Errores NO RECUPERABLES detectados"
  description: |
    Tasa: {{ $value }} non-recoverable errors/sec
    Estos NO se reintentarán, NO hay fallback
    
    Causas típicas:
    - CodeUnauthorized: Token expirado, permisos incorrectos
    - CodeInvalidInput: Usuario pasó datos inválidos
    - CodeMemoryError: Bug en aplicación
  action: "Revisar logs para identificar bug o config incorrecta"
```

**Cuándo dispara**:
- Más de 1 error no-recuperable por segundo durante 2+ minutos

**Acciones**:
1. ✅ Ver qué tipo de error no-recuperable: UNAUTHORIZED? INVALID_INPUT? MEMORY_ERROR?
2. ✅ Investigar causa raíz
3. ✅ Esto indica un bug de aplicación o misconfigración

**Remediación**:
```bash
# Ver qué específicamente falla
kubectl logs <pod> | grep "non-recoverable\|UNAUTHORIZED\|INVALID_INPUT"

# Ejemplos:
# UNAUTHORIZED → Revisar tokens en config
# INVALID_INPUT → Validar entrada de usuario
# MEMORY_ERROR → Investigar memory leak
```

---

## Ejemplos de Uso

### Ejemplo 1: Instrumentar un Servicio

```go
package main

import (
    "context"
    "log/slog"
    
    "github.com/jllopis/kairos/pkg/errors"
    "github.com/jllopis/kairos/pkg/resilience"
    "github.com/jllopis/kairos/pkg/telemetry"
    "go.opentelemetry.io/otel"
)

func main() {
    // 1. Inicializar telemetría
    shutdown, _ := telemetry.Init("my-service", "1.0.0")
    defer shutdown(context.Background())
    
    // 2. Crear métricas
    metrics, _ := telemetry.NewErrorMetrics(context.Background())
    
    ctx := context.Background()
    tracer := otel.Tracer("my-service")
    
    // 3. En tu función de negocio
    _, span := tracer.Start(ctx, "ProcessRequest")
    defer span.End()
    
    // 4. Llamar a servicio con reintentos
    retryConfig := resilience.DefaultRetryConfig().
        WithMaxAttempts(3).
        WithInitialDelay(100 * time.Millisecond)
    
    err := retryConfig.Do(ctx, func() error {
        return callLLM(ctx)
    })
    
    // 5. Registrar resultado
    if err != nil {
        // Error no recuperable
        metrics.RecordErrorMetric(ctx, err, "llm-service")
        telemetry.RecordError(span, err)
        slog.Error("LLM failed", "error", err)
        return err
    }
    
    // Éxito
    metrics.RecordRecovery(ctx, errors.CodeLLMError)
    return nil
}

func callLLM(ctx context.Context) error {
    // Si falla con error recuperable → retry
    // Si falla con error no-recuperable → RecordErrorMetric
    return errors.New(
        errors.CodeLLMError,
        "model overloaded",
        nil,
    ).WithRecoverable(true)
}
```

**Resultado en dashboards**:
- Counter `kairos.errors.total{error_code="LLM_ERROR", component="llm-service"}` incrementa en cada intento
- Counter `kairos.errors.recovered{error_code="LLM_ERROR"}` incrementa si retry funciona
- Tasa de recuperación: (1 / 3 intentos) = 33% si falla después de 3 reintentos

---

### Ejemplo 2: Consultar Métricas en Grafana

**Dashboard: "Top Failing Components"**

```promql
# Muestra componentes con más errores
topk(5, sum(rate(kairos.errors.total[5m])) by (component))
```

Resultado:
```
llm-service:       2.5 errors/sec
executor:          0.8 errors/sec
cache:             0.2 errors/sec
database:          0.1 errors/sec
```

**Action**: Investigar `llm-service` primero.

---

### Ejemplo 3: Interpretar una Cascada de Fallos

**Timeline**:
```
T=10:00:00 → external-api.com sufre latencia (lento)
T=10:00:15 → kairos.errors.total{error_code="TIMEOUT"} sube
T=10:00:30 → kairos.circuitbreaker.state{component="api-client"} = 0 (OPEN)
T=10:00:35 → kairos.health.status{component="llm-service"} = 1 (DEGRADED)
T=10:00:45 → kairos.errors.rate{component="llm-service"} sube
T=10:01:00 → AlertManager dispara KairosCircuitBreakerOpen + KairosHighErrorRate
T=10:02:00 → external-api.com recuperada
T=10:02:15 → kairos.circuitbreaker.state retorna a 2 (CLOSED)
T=10:02:30 → kairos.health.status vuelve a 2 (HEALTHY)
```

**Insight**: Un timeout en un servicio externo cascadeó a través del sistema. Circuit breaker + health checks evitó outage total.

---

## Integración con Backends

### Datadog

```yaml
# datadog-agent-helm-values.yaml
datadog:
  apiKey: <API_KEY>
  otlp:
    enabled: true
    receiver:
      protocols:
        grpc:
          enabled: true
          port: 4317
        http:
          enabled: true
          port: 4318
```

**Configurar Kairos**:
```go
metrics, _ := telemetry.NewErrorMetrics(ctx)
shutdown, _ := telemetry.InitWithConfig(
    "my-service", "1.0.0",
    telemetry.Config{
        Exporter:     "otlp",
        OTLPEndpoint: "datadog-agent:4317",
    },
)
```

### New Relic

```go
shutdown, _ := telemetry.InitWithConfig(
    "my-service", "1.0.0",
    telemetry.Config{
        Exporter:     "otlp",
        OTLPEndpoint: "otlp.nr-data.net:4317",
    },
)
```

Luego configurar API key en variable de entorno `OTEL_EXPORTER_OTLP_HEADERS`.

### Prometheus + Grafana (On-Premise)

```yaml
# prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'kairos'
    static_configs:
      - targets: ['localhost:8888']  # OTEL Prometheus exporter
```

**Deploy**:
```bash
docker run -p 9090:9090 prom/prometheus --config.file=prometheus.yml
docker run -p 3000:3000 grafana/grafana
```

---

## SLOs y Recomendaciones

### SLO 1: Error Rate

**Meta**: < 5 errores/min (99.9% availability a 1M req/min)

```promql
# Alertar si se excede durante 10 min
avg(rate(kairos.errors.total[5m])) over (10m) > 5
```

**Acciones**:
- Escalar automáticamente si es sobrecarga
- Investigar si hay bug nuevo

---

### SLO 2: Recovery Rate

**Meta**: > 80% de errores se recuperan

```promql
# Alertar si recovery rate cae
(
  avg(rate(kairos.errors.recovered[5m])) over (10m)
  /
  avg(rate(kairos.errors.total[5m])) over (10m)
) < 0.8
```

**Acciones**:
- Aumentar MaxAttempts en RetryConfig
- Activar fallback strategies
- Revisar qué errores son non-recoverable

---

### SLO 3: Component Health

**Meta**: Todos los componentes HEALTHY >= 95% del tiempo

```promql
# Calcular uptime por componente
(
  sum(increase(kairos.health.status{component=~".+", status="HEALTHY"}[1d]))
  /
  sum(increase(kairos.health.status{component=~".+"}[1d]))
) * 100
```

---

### Recomendaciones Generales

1. **Baselines**: Establece baselines para tu servicio (error rate normal, recovery rate esperada)
2. **Tunning**: Ajusta thresholds de alertas basado en tu SLO
3. **Runbooks**: Para cada alerta, tener runbook de remediación
4. **Correlación**: Buscar patrones → ¿error A siempre dispara error B?
5. **Costo**: Monitorea CodeRateLimit para optimizar capacity planning

---

## Referencias

- [Manejo de Errores en Kairos](ERROR_HANDLING.md)
- [Documentación OTEL](https://opentelemetry.io/)
- [Prometheus Query Language](https://prometheus.io/docs/prometheus/latest/querying/basics/)
- [Grafana Dashboard Best Practices](https://grafana.com/docs/grafana/latest/dashboards/)
