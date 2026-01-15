# Arquitectura - Kairos

## Objetivos

- Runtime Go-first con SDK de primera clase.
- Interoperabilidad por estándares: MCP, A2A/ACP, AgentSkills, AGENTS.md.
- Observabilidad por defecto con OpenTelemetry (trazas, métricas y logs).
- Ejecución multiagente distribuida con governance y foco en producción.

## Arquitectura por capas

1. Interfaces (UI/CLI)
2. Control Plane (API, auth, políticas, governance)
3. Runtime multiagente (Go)
4. Planner + Memoria + Tools
5. Interop: MCP + A2A/ACP + AGENTS.md
6. Observabilidad + Storage

## Componentes core (Go)

- Runtime de agente: ciclo de vida, scheduling, contexto, ejecución de tools.
- Planner: grafos explícitos y planner emergente con un modelo interno común.
- Memoria: corta, larga, compartida, persistente.
- Tools/Skills: AgentSkills como capa semántica; binding a tools MCP.
- Policy engine: scopes, allow/deny y eventos de auditoría.

## Interoperabilidad

- Cliente y servidor MCP.
- A2A/ACP para discovery, delegación y ejecución remota.
- AGENTS.md cargado automáticamente al inicio para aplicar reglas del repo.

## Integraciones enterprise

La idea es que las integraciones con APIs externas y sistemas corporativos
puedan declararse de forma estándar. En la práctica, esto encaja con MCP como
puente de tools y con especificaciones tipo OpenAPI para describir servicios.

## Depuración y control visual

Además del CLI, el objetivo es una interfaz visual para ver flujos, trazas y
estado en tiempo real, con capacidad de intervenir cuando haga falta.

### Implementación A2A (estado actual)

- Binding gRPC con streaming (SendMessage, SendStreamingMessage, GetTask, ListTasks, CancelTask).
- Tipos Go generados desde `pkg/a2a/proto/a2a.proto` (`scripts/gen-a2a.sh`).
- Publicación de AgentCard + discovery, con servidor/cliente A2AService.
- Mapeo de Task/Message/Artifact con respuestas por streaming.
- Bindings HTTP+JSON y JSON-RPC (`pkg/a2a/httpjson`, `pkg/a2a/jsonrpc`) con SSE.
- Task store + push config store (in-memory + SQLite).
- Demo multiagente (demoKairos) con delegación (orchestrator -> knowledge/spreadsheet).

### Backends de almacenamiento A2A

- Stores in-memory: `MemoryTaskStore`, `MemoryPushConfigStore` (por defecto en handlers).
- Stores SQLite (sin CGO): `SQLiteTaskStore`, `SQLitePushConfigStore` via `modernc.org/sqlite`.
- Esquema creado al inicio; tasks/configs como JSON con índices por estado, contexto y update time.
- Paginación con orden estable: `updated_at DESC`, luego `id ASC`.

## Observabilidad

- Trazas OpenTelemetry para ejecuciones de agente, pasos del planner, tools y hops A2A.
- Métricas: latencia por paso, errores por agente, uso de tokens.
- Logs estructurados con ids de trace/span y resúmenes de decisión.
- Eventos por iteración, incluyendo resultados de tool calls para auditoría.
- **Error Handling**: Typed errors con clasificación automática, retry, circuit breaker
- **Production Monitoring**: 5 OTEL metrics, 6 alert rules, 3 dashboards

**Para más detalles sobre error handling y observabilidad:**
- Ver [Error Handling Guide](ERROR_HANDLING.md) (Phase 1-3 complete)
- Ver [Observability Guide](OBSERVABILITY.md) (dashboards, alerts, SLOs)

### Configuración de telemetría (OTLP)

Ejemplo de config para exporter OTLP:

```json
{
  "telemetry": {
    "exporter": "otlp",
    "otlp_endpoint": "localhost:4317",
    "otlp_insecure": true
  }
}
```

Variables de entorno equivalentes:

- `KAIROS_TELEMETRY_EXPORTER`
- `KAIROS_TELEMETRY_OTLP_ENDPOINT`
- `KAIROS_TELEMETRY_OTLP_INSECURE`
- `KAIROS_TELEMETRY_OTLP_TIMEOUT_SECONDS`

#### Verificación

1) Levanta un backend OTLP compatible (por ejemplo `localhost:4317`).
2) Ejecuta un ejemplo con OTLP habilitado:

```bash
KAIROS_TELEMETRY_EXPORTER=otlp \
KAIROS_TELEMETRY_OTLP_ENDPOINT=localhost:4317 \
KAIROS_TELEMETRY_OTLP_INSECURE=true \
go run ./examples/basic-agent
```

3) Confirma que las trazas y métricas llegan al backend.

Smoke test opcional:

```bash
KAIROS_OTLP_SMOKE_TEST=1 \
KAIROS_TELEMETRY_OTLP_ENDPOINT=localhost:4317 \
KAIROS_TELEMETRY_OTLP_INSECURE=true \
KAIROS_TELEMETRY_OTLP_TIMEOUT_SECONDS=30 \
go test ./pkg/telemetry -run TestOTLPSmoke -count=1
```

## Modelo de datos (alto nivel)

- Agent: id, role, skills, tools, memory, policies.
- Skill: capacidad semántica (AgentSkills spec).
- Tool: implementación MCP que cumple una skill.
- Plan: grafo o estado emergente.
- Memory: interfaz Store/Retrieve con backends plugables.

## Base del planner explícito

- Esquema de grafos (`pkg/planner`): nodos, edges y start opcional.
- Parsers JSON/YAML con validación.
- Executor con trazas por nodo, branching y evaluación multi-edge.

## Flujo de ejecución (runtime)

1) Cargar AGENTS.md y aplicar reglas del repo.
2) Inicializar agente con skills, memoria, tools y políticas.
3) Construir plan (grafo explícito o emergente).
4) Ejecutar pasos con propagación de contexto.
5) Emitir trazas/métricas/logs y eventos de auditoría.

## Fuentes de configuración

- Archivo: `~/.kairos/settings.json` o `./.kairos/settings.json`.
- Env: `KAIROS_*` (mapea a keys de config).
- CLI: `--config=/ruta/a/settings.json` y `--set key=value` (repetible).

Precedencia: valores por defecto < archivo < env < CLI.

Ejemplo:

```bash
go run ./examples/basic-agent --config=./.kairos/settings.json \
  --set llm.provider=ollama \
  --set telemetry.exporter=stdout
```

Ver `docs/CONFIGURATION.md` para la guía completa.

## Taxonomía de eventos

Eventos semánticos para streaming/logs: `docs/EVENT_TAXONOMY.md`.

## Tasks

API core de Task y ciclo de vida: `docs/TASKS.md`.

## Discovery

Patrones de discovery: `docs/protocols/A2A/topics/agent-discovery.md`.

## Opciones del agent loop

- `agent.WithDisableActionFallback(true)` desactiva el parsing legacy "Action:".
- `agent.WithActionFallbackWarning(true)` emite un aviso cuando se usa el fallback.
- Config: `agent.disable_action_fallback` o `KAIROS_AGENT_DISABLE_ACTION_FALLBACK=true` (por defecto: true).
- Sobrescrituras por agente bajo `agents.<agent_id>`.

### Plan de deprecación del fallback

- Fase 1 (actual): fallback desactivado por defecto; habilitar explícitamente.
- Fase 2 (siguiente minor): aviso en cada uso + nota en docs/changelog.
- Fase 3 (siguiente minor): requiere flag explícito y aviso al arranque.
- Fase 4 (siguiente major): eliminar fallback y flags asociados.

Activación:

- Habilita fallback solo con `agent.disable_action_fallback=false`.

Ejemplo de config:

```json
{
  "agent": {
    "disable_action_fallback": false,
    "warn_on_action_fallback": true
  },
  "agents": {
    "mcp-agent": {
      "disable_action_fallback": true
    }
  }
}
```

Ejemplo completo con telemetría:

```json
{
  "agent": {
    "disable_action_fallback": false
  },
  "agents": {
    "mcp-agent": {
      "disable_action_fallback": true
    }
  },
  "telemetry": {
    "exporter": "otlp",
    "otlp_endpoint": "localhost:4317",
    "otlp_insecure": true
  }
}
```

## Governance y seguridad

- Enforcement de políticas en tools y delegación.
- Puntos human-in-the-loop.
- Auditoría de cada acción y tool call.

## Despliegue

- Binario Go único.
- Docker/Kubernetes ready.
- Escalado horizontal con federación A2A.

## Layout de paquetes (inicial)

- core/agent
- core/runtime
- core/planner
- core/memory
- core/tools
- interop/mcp
- interop/a2a
- observability/otel
- controlplane/api
- ui (future)

## Enlaces relacionados

- Planner: `docs/Conceptos_Planner.md`
- Demo: `docs/Demo_Kairos.md`
- A2A bindings: `docs/protocols/A2A/topics/bindings.md`
- Governance: `docs/governance-usage.md`
