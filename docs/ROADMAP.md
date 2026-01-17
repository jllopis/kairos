# Kairos Roadmap

## Estado del Proyecto

Kairos es un framework de agentes IA en Go, **production-ready** con las siguientes capacidades:

| Área | Estado | Descripción |
|------|--------|-------------|
| Core Runtime | ✅ Completo | Agent loop, context propagation, lifecycle management |
| MCP Protocol | ✅ Completo | Client/server, stdio/HTTP, tool binding |
| A2A Protocol | ✅ Completo | gRPC, HTTP+JSON, JSON-RPC, discovery |
| Observability | ✅ Completo | OTLP traces/metrics, structured logs |
| Planners | ✅ Completo | Explicit (graph) + Emergent (ReAct) |
| Memory | ✅ Completo | In-memory, file, vector store |
| Governance | ✅ Completo | Policies, AGENTS.md, HITL, audit |
| LLM Providers | ✅ Completo | Ollama, OpenAI, Anthropic, Gemini, Qwen |
| CLI | ✅ Completo | init, run, validate, explain, graph |
| Streaming | ✅ Completo | Real-time responses para OpenAI/Anthropic/Gemini |
| Connectors | ✅ Completo | OpenAPI → tools automáticos |
| Security | ✅ Completo | Guardrails, PII filtering, prompt injection |
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
- **Streaming**: Respuestas en tiempo real
- **OpenAPI Connector**: Convierte cualquier API REST en tools
- **Guardrails**: Prompt injection, PII filtering, content filtering
- **Testing Framework**: Escenarios, mocks, assertions
- **Hot-reload**: `kairos run --watch`

---

## Próximos Pasos

### Prioridad Alta 🔴

| Feature | Descripción | Estado |
|---------|-------------|--------|
| UI Visual | Timeline de ejecución, inspector de memoria | Planificado |
| Skill Marketplace | Registry de skills compartidos | Planificado |

### Prioridad Media 🟡

| Feature | Descripción | Estado |
|---------|-------------|--------|
| GraphQL Connector | Similar a OpenAPI pero para GraphQL | Planificado |
| Streaming Qwen/Ollama | Completar streaming en todos los providers | Planificado |

### Largo Plazo 🟢

| Feature | Descripción | Estado |
|---------|-------------|--------|
| kairosctl MVP | Control plane, workflow store, agent registry | Planificado |
| kairosctl Avanzado | Scheduler, queue distribuida, editor visual | Planificado |

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
| Ollama | `pkg/llm/ollama.go` | `llama3` | 🚧 |
| OpenAI | `providers/openai/` | `gpt-5-mini` | ✅ |
| Anthropic | `providers/anthropic/` | `claude-haiku-4` | ✅ |
| Gemini | `providers/gemini/` | `gemini-3-flash-preview` | ✅ |
| Qwen | `providers/qwen/` | `qwen-turbo` | 🚧 |

Ver [PROVIDERS.md](PROVIDERS.md) para documentación completa.

---

## Arquitectura de kairosctl (Futuro)

Plataforma de orquestación estilo n8n para workflows y agentes.

**Decisión de arquitectura:**
- Dos repositorios: `kairos` (biblioteca) + `kairosctl` (orquestador)
- `kairosctl` importa `kairos` como dependencia Go
- Kairos mantiene su rol de framework
- kairosctl añade: scheduling, persistence, registry, UI visual

**Interfaces estables de Kairos:**
- `core.Agent`, `core.Task`, `core.Skill`
- `llm.Provider`, `llm.StreamingProvider`
- `a2a.Client`
- `planner.Executor`
- `core.EventEmitter`

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
| [MCP](protocols/MCP.md) | Model Context Protocol |
| [A2A](protocols/A2A/Overview.md) | Agent-to-Agent Protocol |

---

## Contribuir

1. Fork el repositorio
2. Crea una rama para tu feature
3. Asegúrate de que los tests pasen: `go test ./...`
4. Envía un PR

Ver [AGENTS.md](/AGENTS.md) para convenciones del proyecto.
