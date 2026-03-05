# Kairos Roadmap

## Estado del Proyecto

Kairos es un framework de agentes IA en Go. La auditoría técnica (2026-01-28) confirma una base amplia y funcional, pero **no es production-ready** todavía: hay gaps de integración y control plane aún por definir.

| Área | Estado | Descripción |
|------|--------|-------------|
| Core Runtime | ✅ Completo | Agent loop base, context propagation, lifecycle management |
| MCP Protocol | ✅ Completo | Client/server, stdio/HTTP, tool binding |
| A2A Protocol | ✅ Completo | gRPC, HTTP+JSON, JSON-RPC, discovery |
| Observability | ✅ Completo | OTLP traces/metrics + logs estructurados con correlación de trace |
| Planners | ✅ Completo | Planner explícito + emergente integrados en runtime |
| Memory | ✅ Completo | In-memory, file, vector store, conversation memory |
| Governance | 🟡 Parcial | Policies y filtros; HITL local integrado |
| LLM Providers | ✅ Completo | Ollama, OpenAI, Anthropic, Gemini, Qwen |
| CLI | ✅ Completo | init, run, validate, explain, graph |
| Streaming | ✅ Completo | Streaming providers (según providers) |
| Connectors | ✅ Completo | OpenAPI, GraphQL, gRPC, SQL |
| Security | 🟡 Parcial | Guardrails integrados vía runtime/CLI; cobertura extensible |
| Testing | ✅ Completo | Scenarios, mock providers, assertions |

---

## Fases Completadas

### Fase 0-4: Foundations ✅
- Core interfaces (Agent, Tool, Skill, Plan, Memory)
- MCP interoperability con retry/timeout/cache
- OTLP traces y metrics
- Explicit planner (YAML/JSON graphs)
- Emergent planner (ReAct loop)

### Fase 5-7: Distributed ✅
- A2A protocol con todos los bindings
- Multi-level memory (short/long term)
- Governance, policies y AGENTS.md

### Fase 8-9: Developer Experience ✅
- CLI completo con scaffolding
- 18 ejemplos progresivos
- Corporate templates (CI/CD, Docker, observability)
- Config layering con perfiles

### Fase 10: Production Features ✅
- **LLM Providers**: OpenAI, Anthropic, Gemini, Qwen como módulos independientes
- **Streaming**: Respuestas en tiempo real (todos los providers)
- **Conectores declarativos**: OpenAPI, GraphQL, gRPC, SQL → tools automáticos
- **Guardrails**: Prompt injection, PII filtering, content filtering
- **Testing Framework**: Escenarios, mocks, assertions
- **Hot-reload**: `kairos run --watch`

---

## Auditoría Técnica (2026-01-28)

Resumen de gaps relevantes (ver “Plan de Acción”):
- Planner explícito no integrado con el loop del agente. (✅ Resuelto)
- HITL local en tool calls no tiene workflow interactivo. (✅ Resuelto)
- Observabilidad con atributos ricos y logs correlados. (✅ Resuelto)
- Guardrails no están “plugged” por defecto en el runtime. (✅ Resuelto)
- Control plane (`kairosctl`) por definir: registries A2A/MCP/Skills, spaces/apps/workflows y ejecución de plataforma.

## Plan de Acción (priorizado)

### Prioridad 0: Integraciones de Runtime 🔴

1) **Planner explícito integrado en runtime**
   - Objetivo: unificar planner explícito + emergente bajo una interfaz común.
   - Entregables:
     - Interfaz `planner.Plan`/`planner.Executor` conectada al loop de `pkg/agent/`.
     - Opción `agent.WithPlanner(...)` + soporte de YAML/JSON.
     - Telemetría y eventos por nodo/edge en el loop.
   - Resultado esperado: mismo agente puede ejecutar flujo declarativo o emergente.
   - Estado: ✅ Completado (2026-01-28)

2) **HITL local en tool calls**
   - Objetivo: cuando policy devuelve `pending`, activar flujo de aprobación interactivo.
   - Entregables:
     - Hook de aprobación local en `agent.Run` (bloqueante o async configurable).
     - UI/CLI simple de approvals en modo local (reuse `pkg/a2a/server/approval_*`).
     - Persistencia configurable (memoria/SQLite) para approvals locales.
   - Resultado esperado: el agente local no responde “Policy denied” cuando es “pending”.
   - Estado: ✅ Completado (2026-01-28)

### Prioridad 1: Observabilidad y Seguridad 🟡

3) **Observabilidad enriquecida**
   - Añadir atributos ricos (tool args/result, memoria, estado interno) de forma consistente.
   - Exportador de logs OTEL o integración de logs estructurados con contexto de trace.
   - Estado: ✅ Completado (2026-01-28)

4) **Guardrails integrados por defecto**
   - Opciones en `agent.New` para activar guardrails en entrada/salida.
   - Configuración vía `config` y CLI.
   - Estado: ✅ Completado (2026-01-28)

### Prioridad 2: Control Plane (`kairosctl`) 🟢

5) **Definición y primer MVP**
   - **Objetivo:** `kairosctl` como plataforma de orquestación, con registros globales y ejecución multi‑tenant (spaces/apps/workflows).
   - Alcance mínimo:
     - Registries: A2A, MCP, Skills globales (versionado + metadatos).
     - Gestión de espacios/apps/workflows + ejecución programada/manual.
     - API y UI básica de operación (estado, histórico, replay).
   - Nota: `kairos` mantiene CLI local; `kairosctl` gestiona plataforma.

## Próximos Pasos

### Prioridad Alta 🔴

| Feature | Descripción | Ubicación |
|---------|-------------|-----------|
| UI Web Configurable | ✅ Completado (2026-01-28) | Kairos |

### Para kairosctl 🟡

| Feature | Descripción | Estado |
|---------|-------------|--------|
| Skill Registry | Registro global de skills versionadas | Planificado |
| A2A Registry | Registro centralizado de agentes A2A | Planificado |
| MCP Registry | Registro de servidores MCP disponibles | Planificado |
| Agent Registry | Catálogo de agentes con versiones | Planificado |
| Spaces/Apps/Workflows | Entidades lógicas para ejecución y gobierno | Planificado |
| Dashboard Visual | Timeline, histórico, replay de ejecuciones | Planificado |

### Largo Plazo 🟢

| Feature | Descripción | Estado |
|---------|-------------|--------|
| kairosctl MVP | Control plane, registries, espacios/apps/workflows | Planificado |
| kairosctl Avanzado | Scheduler, cola distribuida, editor visual | Planificado |

---

## UI Web Existente (`kairos web`)

El CLI incluye una UI web básica para desarrollo local:

```bash
kairos web --addr :8088
```

Endpoints disponibles:
- `/agents` - Lista de agentes A2A descubiertos
- `/tasks` - Gestión de tareas (list, detail, stream)
- `/approvals` - Human-in-the-loop approvals

**Nota:** Para producción y funcionalidades avanzadas (histórico, métricas, registries), usar kairosctl.

---

## LLM Providers

Arquitectura de módulos Go independientes para importación selectiva:

```bash
# Solo instala lo que necesites
go get github.com/jllopis/kairos/providers/openai
go get github.com/jllopis/kairos/providers/anthropic
go get github.com/jllopis/kairos/providers/gemini
go get github.com/jllopis/kairos/providers/qwen
```

| Provider | Módulo | Default Model | Streaming |
|----------|--------|---------------|-----------|
| Ollama | `pkg/llm/ollama.go` | `llama3` | ✅ |
| OpenAI | `providers/openai/` | `gpt-5-mini` | ✅ |
| Anthropic | `providers/anthropic/` | `claude-haiku-4` | ✅ |
| Gemini | `providers/gemini/` | `gemini-3-flash-preview` | ✅ |
| Qwen | `providers/qwen/` | `qwen-turbo` | ✅ |

Ver [PROVIDERS.md](PROVIDERS.md) para documentación completa.

---

## Arquitectura de kairosctl

Plataforma de operación de agentes IA estructurada en tres pilares:
**Observability**, **Evaluation** y **Deployment & Orchestration**.

> La visión completa, el alcance detallado, las fases de implementación y la
> comparativa con plataformas de referencia están documentados en el repositorio
> de kairosctl:
> - **[kairosctl/docs/VISION.md](https://github.com/jllopis/kairosctl/blob/main/docs/VISION.md)** — Visión, pilares y plan de evolución.
> - **[kairosctl/docs/ROADMAP.md](https://github.com/jllopis/kairosctl/blob/main/docs/ROADMAP.md)** — Roadmap operativo con fases y criterios.

**Decisión de arquitectura:**
- Dos repositorios: `kairos` (framework) + `kairosctl` (control plane).
- `kairosctl` importa `kairos` como dependencia Go.
- Kairos mantiene su rol de biblioteca/framework.
- kairosctl añade: observabilidad (trazas, dashboards, alertas), evaluación
  (datasets, evals offline/online, human feedback), y orquestación (registries,
  scheduler, task queues, multi-tenant, RBAC).

**Pilares de kairosctl:**

| Pilar | Capacidades clave |
|-------|-------------------|
| **Observability** | Receptor OTLP, trace explorer, dashboards (latencia, coste, error rate), alertas, insights |
| **Evaluation** | Datasets, offline/online evals, LLM-as-judge, human feedback, comparador side-by-side, CI integration |
| **Deployment & Orchestration** | Agent/Skill/MCP registries con versionado, scheduler cron, task queues, HITL, multi-tenant (RBAC), audit log |

**Interfaces estables de Kairos (contrato con kairosctl):**
- `core.Agent`, `core.Task`, `core.Skill`
- `llm.Provider`, `llm.StreamingProvider`
- `a2a.Client`
- `planner.Executor`
- `core.EventEmitter`
- `memory.ConversationMemory`

**Política de compatibilidad:** Kairos sigue SemVer. kairosctl pinea una
versión mínima en `go.mod`. Cambios breaking requieren PR coordinado en ambos
repos.

---

## Ejemplos Disponibles

| # | Ejemplo | Descripción |
|---|---------|-------------|
| 01 | hello-agent | Agente mínimo |
| 02 | mcp-tools | Tools via MCP |
| 03 | observability | OTLP tracing |
| 04 | explicit-plan | YAML/JSON graphs |
| 05 | emergent-plan | ReAct loop |
| 06 | a2a-multi | Multi-agent A2A |
| 07 | memory | Short/long term |
| 08 | governance | Policies y audit |
| 09 | streaming-events | Event streaming |
| 10 | config-layering | Perfiles dev/prod |
| 11 | mcp-pool | Connection pooling |
| 12 | error-handling | Retry, circuit breaker |
| 13 | cli-integration | CLI commands |
| 14 | guardrails | Security filters |
| 15 | testing | Test scenarios |
| 16 | providers | LLM auth/tokens |
| 17 | openapi-connector | REST → tools |
| 18 | streaming | Real-time responses |
| 19 | graphql | GraphQL tools |
| 20 | conversation-memory | Multi-turn conversations |

---

## Documentación

| Documento | Descripción |
|-----------|-------------|
| [Inicio Rápido](Inicio_Rapido.md) | 5 minutos para tu primer agente |
| [Arquitectura](ARCHITECTURE.md) | Diseño del sistema |
| [CLI](CLI.md) | Comandos disponibles |
| [Configuración](CONFIGURATION.md) | Config layering, perfiles |
| [Providers](PROVIDERS.md) | OpenAI, Anthropic, Gemini, Qwen |
| [Guardrails](GUARDRAILS.md) | Seguridad y filtros |
| [Testing](TESTING.md) | Framework de testing |
| [Skills](Skills.md) | AgentSkills specification |
| [Conversation Memory](CONVERSATION_MEMORY.md) | Multi-turn chat history |
| [MCP](protocols/MCP.md) | Model Context Protocol |
| [A2A](protocols/A2A/Overview.md) | Agent-to-Agent Protocol |

---

## Contribuir

1. Fork el repositorio
2. Crea una rama para tu feature
3. Asegúrate de que los tests pasen: `go test ./...`
4. Envía un PR

Ver [AGENTS.md](/AGENTS.md) para convenciones del proyecto.
