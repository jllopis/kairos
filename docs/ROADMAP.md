# Roadmap y fases

## Leyenda de estado

- [ ] Planificado
- [~] En progreso
- [x] Hecho

## Hitos

- M0: Go SDK skeleton + hello agent (Fase 0) [x]
- M1: Agent can call external MCP tool (Fase 1) [x]
- M2: OTel traces visible in backend (Fase 2) [x]
- M3: YAML/JSON graph executes end-to-end (Fase 3) [x]
- M4: Emergent flow runs with decision logs (Fase 4) [x]
- M5: Two agents delegate with distributed traces (Fase 5) [x]
- M6: Per-agent memory with short/long backends (Fase 6) [x]
- M7: AGENTS.md and policies enforced with audit trail (Fase 7) [x]
- M8: Operator UI with agents, flows, and traces (Fase 8) [x]
- M9: Developer Experience improvements (Fase 9 - DX) [x]

## Fase 0: Core foundations (Estimación: M)
Objetivos: core interfaces and minimal runtime.
Dependencias: none.
Hito: Go SDK skeleton + hello agent.
Tareas:

- [x] Define core interfaces (Agent, Tool, Skill, Plan, Memory).
- [x] Create runtime lifecycle (start, run, stop).
- [x] Add minimal context propagation support.
- [x] Provide hello agent example.
- [x] Align public agent options with examples (model selection).
Aceptación: US-01 (partial), US-06 (interface).

## Fase 1: MCP interoperability (Estimación: M)
Objetivos: MCP client/server and tool binding.
Dependencias: Fase 0.
Hito: Agent can call external MCP tool.
Tareas:

- [x] MCP client with tool invocation.
- [x] MCP server for exposing tools.
- [x] Skill -> MCP tool binding (Agent tool adapter).
- [x] Tool schema/arguments mapping and validation.
- [x] Example: MCP agent loads tools from config and runs a tool call.
- [x] Error handling, retries, and timeout policy for tool calls.
- [x] Tool discovery caching and refresh strategy.
Pendientes (post-hito):

- [x] Expose MCP retry/timeout/cache policy in config.
- [x] Add end-to-end MCP smoke tests for stdio + HTTP.
Aceptación: US-01 complete.
Notas:

- Core MCP path works via config + stdio/http client/server; hardening now defaults in the client.
- MCP retry/timeout/cache policy is configurable in `mcp.servers` settings.

## Fase 2: Observability baseline (Estimación: M)
Objetivos: OTel traces/metrics/logs from runtime.
Dependencias: Fase 0.
Hito: traces visible in OTel backend.
Tareas:

- [x] OTel tracer and span propagation in runtime (Agent/Runtime + tool/memory/LLM spans).
- [x] Metrics for latency and error counts (stdout exporter).
- [x] Structured logs with trace/span ids.
- [x] Configurable OTLP exporter (traces + metrics) and resource attributes.
- [x] Example config for OTLP backend.
- [x] Validate OTLP export against a backend and document a smoke-test.
Aceptación: US-05 (partial).
Notas:

- OTLP smoke test is opt-in via environment variables to avoid default test dependencies.

## Fase 3: Explicit planner (Estimación: L)
Objetivos: deterministic graph execution.
Dependencias: Fase 0, Fase 1.
Hito: YAML/JSON graph executes end-to-end.
Tareas:

- [x] Graph model and executor.
- [x] YAML/JSON parser and serializer.
- [x] Per-node tracing (spans).
- [x] Audit events for node execution.
- [x] Documentation and example for graph usage.
Pendientes (post-hito):

- [x] Branching/conditions and multi-edge evaluation.
- [x] Graph serialization helpers (emit JSON/YAML).
Aceptación: US-03 complete.

## Fase 4: Emergent planner (Estimación: M)
Objetivos: dynamic next-step decisions.
Dependencias: Fase 0, Fase 1.
Hito: emergent flow runs with decision logs.
Tareas:

- [x] Decision engine with tool selection (basic ReAct loop).
- [x] Logging of decisions and outcomes (decision rationale + inputs/outputs).
- [x] Structured tool-call parsing (LLM tool calls + JSON args).
- [x] Provide tool definitions to LLM (function schema) for native tool calls.
- [x] Prefer tool calls over "Action:" parsing when available (deprecate string parsing path).
- [x] Optional warning when legacy "Action:" parsing is used.
Aceptación: US-04 complete.
Notas:

- Fallback "Action:" parsing is configurable and disabled by default; deprecation path remains documented.

## Fase 5: A2A distributed runtime (Estimación: L)
Objetivos: discovery, delegation, and trace continuity.
Dependencias: Fase 0, Fase 2.
Hito: two agents delegate with distributed traces.
Tareas:

- [x] Pin A2A proto version and generate Go types from `pkg/a2a/proto/a2a.proto`.
- [x] Implement gRPC binding with streaming (SendMessage, SendStreamingMessage, GetTask, ListTasks, CancelTask).
- [x] AgentCard publishing (well-known) + discovery client.
- [x] Remote agent invocation (call/response) and task lifecycle mapping.
- [x] Trace context propagation over A2A (end-to-end).
- [x] Minimal auth middleware hooks (OIDC/mTLS stubs; config-driven).
- [x] Conformance tests (golden proto/JSON payloads, streaming order, cancel).
- [x] HTTP+JSON and JSON-RPC bindings.
- [x] Implement ListTasks pagination with page tokens.
- [x] SQLite-backed TaskStore/PushConfigStore (no CGO) for persistence.
- [x] Planner-driven multi-agent demo (demoKairos) with A2A + MCP + OTLP.
- [x] Agent discovery patterns (Config / WellKnown / Registry) with configurable order.
Aceptación: US-02 (MVP) complete with trace continuity.
Notas:

- MVP binding is gRPC-first for streaming stability; HTTP+JSON/JSON-RPC server bindings are implemented.
- HTTP+JSON/JSON-RPC client helpers are available for parity with server bindings.
- Demo feedback:
  - Add a bootstrap helper for agents (config + telemetry + llm + mcp) to reduce boilerplate.
  - Provide a lightweight in-process MCP server helper for tool-only agents.
  - Ship a planner-driven multi-agent demo template (A2A + MCP + OTLP) as reference.
  - Add explicit run docs and minimal examples for manual debugging.

## Fase 6: Multi-level memory (Estimación: M)
Objetivos: short and long-term memory backends.
Dependencias: Fase 0.
Hito: per-agent memory configuration.
Tareas:

- [x] In-memory backend.
- [x] Persistent backend (file store, vector store).
- [x] Configuration per agent.
- [x] Agent loop reads/writes memory in runtime.
Aceptación: US-06 complete.

## Fase 7: Governance and AGENTS.md (Estimación: M)
Objetivos: policy enforcement and AGENTS.md loading.
Dependencias: Fase 0, Fase 1.
Hito: policy and AGENTS.md rules enforced.
Tareas:

- [x] AGENTS.md loader and parser.
- [x] Policy engine (scopes, allow/deny).
- [x] Audit event store.
Pendientes (post-hito):

- [x] Config-driven policy rule loading.
- [x] Policy enforcement for A2A/MCP calls (beyond tool gating).
- [x] Server-side policy enforcement for A2A handlers.
- [x] Human-in-the-loop policy flow (approvals + endpoints).
Aceptación: US-07 complete; US-08 complete.

## Fase 8: UI/CLI (Estimación: L)
Objetivos: operator visibility and control.
Dependencias: Fase 2, Fase 5.
Hito: dashboard with agents and traces.
Tareas:

- [x] Fase 8.1 CLI MVP: status, agents, tasks, approvals, mcp list, streaming follow, JSON output.
- [x] Fase 8.2 CLI advanced: traces tail, retry/cancel tasks, approvals tail, export events.
- [x] Fase 8.3 UI skeleton: Agents/Tasks/Approvals/Traces screens and endpoint wiring.
- [x] Fase 8.4 UI operational: streaming, filters, history, audit trail.
Aceptación: US-09 complete.
Notas:

- Fase 8.1 CLI MVP completed (`cmd/kairos`).
- Fase 8.2 CLI advanced completed (traces/tasks/approvals + export).
- Fase 8.3 UI skeleton completed (`cmd/kairos --web`).

## Línea UX core (biblioteca + demo)
Objetivos: make Kairos approachable to developers from Python agent frameworks while keeping standards.
Dependencias: Fase 5, Fase 7.
Tareas core (biblioteca):

- [x] Role/manifest metadata API (coexists with AgentCard).
- [x] Task entity in core with traceable IDs/status/result (no proto/store changes).
- [x] Event taxonomy for semantic streaming/logs (stable types + minimal fields).
Tareas demo:

- [x] Role YAML files to feed core role metadata (`demoKairos/docs/role-*.yaml`).
- [x] Narrative guide: “what it is / what it is not” (`docs/internal/NARRATIVE_GUIDE.md`).
- [x] Demo builder facade (`NewSystem` + `WithAgent` + `WithFlow`), revisit for core after Task/Role/Event stabilize.
- [x] Single entrypoint script for running demo.
Notas:

- No changes to A2A proto or stores in this track.
- See `docs/legacy/walkthrough-demo-improvements.md` for the detailed plan.

## Fase 9: Developer Experience (DX) (Estimación: L)
Objetivos: reducir fricción para nuevos usuarios y equipos enterprise.
Dependencias: Fase 8.
Hito: time-to-first-agent < 5 minutos.
Tareas:

- [x] Reorganizar examples con numeración progresiva (01-13).
- [x] README.md en cada ejemplo con objetivos de aprendizaje.
- [x] CLI scaffolding: `kairos init` con arquetipos (assistant, tool-agent, coordinator, policy-heavy).
- [x] CLI operativo: `kairos run`, `kairos validate`.
- [x] CLI introspección: `kairos explain`, `kairos graph`, `kairos adapters`.
- [x] MCP Connection Pool para escenarios multi-agente (`pkg/mcp/pool`).
- [x] Config layering con perfiles de entorno (`--profile dev|prod`).
- [x] Corporate templates: CI/CD, Dockerfile, docker-compose, observability stack.
- [x] Documentación: CORPORATE_TEMPLATES.md, actualización de CLI.md y CONFIGURATION.md.
Aceptación: DX_PLAN.md complete.
Notas:

- Ver `docs/internal/DX_PLAN.md` para el plan detallado.
- Corporate templates incluyen GitHub Actions, OTEL Collector, Prometheus, Grafana.

## Pendientes de visión (Fase 10+)

### Prioritarios (próximos hitos)

- [x] **Guardrails de seguridad**: prompt injection detection, PII filtering, content filtering (`pkg/guardrails`).
- [ ] **Testing framework**: banco de pruebas y simulación de agentes/flows antes de producción.
- [ ] **Hot-reload de configuración**: `kairos run` con watch mode para desarrollo.

### Medio plazo

- [ ] **Conectores declarativos**: OpenAPI → tools automáticos, GraphQL adapter.
- [ ] **UI visual para debugging**: timeline de ejecución, inspector de memoria, trace explorer.
- [ ] **Skill marketplace**: registry de skills compartidos con versionado.

### Largo plazo (kairosctl)

- [ ] **kairosctl MVP**: repo separado, control plane API, workflow store, agent registry.
- [ ] **kairosctl Avanzado**: scheduler, queue distribuida, editor visual de workflows.

## kairosctl - Orquestador (Proceso Separado - Post M9)

Plataforma de orquestación estilo n8n para workflows, agentes e interacciones LLM.
Ver diseño completo en `docs/internal/ORCHESTRATION_PLATFORM.md`.

Decisión de arquitectura:
- **Dos repositorios**: `kairos` (biblioteca) + `kairosctl` (orquestador).
- `kairosctl` importa `kairos` como dependencia Go.
- Kairos mantiene su rol de framework (runtime, A2A, planner, governance).
- kairosctl añade: scheduling, workflow persistence, registry centralizado, UI visual.

Interfaces a mantener estables en Kairos (forward compatibility):
- `core.Agent`, `core.Task`, `core.Skill` (estructuras core)
- `llm.Provider` (Complete, CompleteWithTools)
- `a2a.Client` (SendMessage, GetTask, ListTasks, CancelTask)
- `planner.Executor` (Execute)
- `core.EventEmitter` (streaming de eventos)

Fases futuras (post M9):
- [ ] kairosctl MVP: repo separado, control plane API, workflow store, agent registry.
- [ ] kairosctl Avanzado: scheduler, queue distribuida, editor visual.

## Dependencias entre hitos (resumen)

- P0 -> P1, P2, P3, P4, P6, P7
- P2 -> P5, P8
- P5 -> P8
- P8 -> P9

## Actualización de seguimiento
Update checkboxes per task and add brief notes under each phase if needed.
See `docs/CONFIGURATION.md` for configuration sources and precedence.
