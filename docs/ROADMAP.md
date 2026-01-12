# Roadmap and Phases

## Status Legend
- [ ] Planned
- [~] In progress
- [x] Done

## Milestones
- M0: Go SDK skeleton + hello agent (Phase 0) [x]
- M1: Agent can call external MCP tool (Phase 1) [x]
- M2: OTel traces visible in backend (Phase 2) [x]
- M3: YAML/JSON graph executes end-to-end (Phase 3) [x]
- M4: Emergent flow runs with decision logs (Phase 4) [x]
- M5: Two agents delegate with distributed traces (Phase 5) [x]
- M6: Per-agent memory with short/long backends (Phase 6) [x]
- M7: AGENTS.md and policies enforced with audit trail (Phase 7) [~]
- M8: Operator UI with agents, flows, and traces (Phase 8) [ ]

## Phase 0: Core foundations (Estimate: M)
Goals: core interfaces and minimal runtime.
Dependencies: none.
Milestone: Go SDK skeleton + hello agent.
Tasks:
- [x] Define core interfaces (Agent, Tool, Skill, Plan, Memory).
- [x] Create runtime lifecycle (start, run, stop).
- [x] Add minimal context propagation support.
- [x] Provide hello agent example.
- [x] Align public agent options with examples (model selection).
Acceptance: US-01 (partial), US-06 (interface).

## Phase 1: MCP interoperability (Estimate: M)
Goals: MCP client/server and tool binding.
Dependencies: Phase 0.
Milestone: Agent can call external MCP tool.
Tasks:
- [x] MCP client with tool invocation.
- [x] MCP server for exposing tools.
- [x] Skill -> MCP tool binding (Agent tool adapter).
- [x] Tool schema/arguments mapping and validation.
- [x] Example: MCP agent loads tools from config and runs a tool call.
- [x] Error handling, retries, and timeout policy for tool calls.
- [x] Tool discovery caching and refresh strategy.
Follow-ups (post-milestone):
- [x] Expose MCP retry/timeout/cache policy in config.
- [x] Add end-to-end MCP smoke tests for stdio + HTTP.
Acceptance: US-01 complete.
Notes:
- Core MCP path works via config + stdio/http client/server; hardening now defaults in the client.
- MCP retry/timeout/cache policy is configurable in `mcp.servers` settings.

## Phase 2: Observability baseline (Estimate: M)
Goals: OTel traces/metrics/logs from runtime.
Dependencies: Phase 0.
Milestone: traces visible in OTel backend.
Tasks:
- [x] OTel tracer and span propagation in runtime (Agent/Runtime + tool/memory/LLM spans).
- [x] Metrics for latency and error counts (stdout exporter).
- [x] Structured logs with trace/span ids.
- [x] Configurable OTLP exporter (traces + metrics) and resource attributes.
- [x] Example config for OTLP backend.
- [x] Validate OTLP export against a backend and document a smoke-test.
Acceptance: US-05 (partial).
Notes:
- OTLP smoke test is opt-in via environment variables to avoid default test dependencies.

## Phase 3: Explicit planner (Estimate: L)
Goals: deterministic graph execution.
Dependencies: Phase 0, Phase 1.
Milestone: YAML/JSON graph executes end-to-end.
Tasks:
- [x] Graph model and executor.
- [x] YAML/JSON parser and serializer.
- [x] Per-node tracing (spans).
- [x] Audit events for node execution.
- [x] Documentation and example for graph usage.
Follow-ups (post-milestone):
- [x] Branching/conditions and multi-edge evaluation.
- [x] Graph serialization helpers (emit JSON/YAML).
Acceptance: US-03 complete.

## Phase 4: Emergent planner (Estimate: M)
Goals: dynamic next-step decisions.
Dependencies: Phase 0, Phase 1.
Milestone: emergent flow runs with decision logs.
Tasks:
- [x] Decision engine with tool selection (basic ReAct loop).
- [x] Logging of decisions and outcomes (decision rationale + inputs/outputs).
- [x] Structured tool-call parsing (LLM tool calls + JSON args).
- [x] Provide tool definitions to LLM (function schema) for native tool calls.
- [x] Prefer tool calls over "Action:" parsing when available (deprecate string parsing path).
- [x] Optional warning when legacy "Action:" parsing is used.
Acceptance: US-04 complete.
Notes:
- Fallback "Action:" parsing is configurable and disabled by default; deprecation path remains documented.

## Phase 5: A2A distributed runtime (Estimate: L)
Goals: discovery, delegation, and trace continuity.
Dependencies: Phase 0, Phase 2.
Milestone: two agents delegate with distributed traces.
Tasks:
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
Acceptance: US-02 (MVP) complete with trace continuity.
Notes:
- MVP binding is gRPC-first for streaming stability; HTTP+JSON/JSON-RPC server bindings are implemented.
 - HTTP+JSON/JSON-RPC client helpers are available for parity with server bindings.
- Demo feedback:
  - Add a bootstrap helper for agents (config + telemetry + llm + mcp) to reduce boilerplate.
  - Provide a lightweight in-process MCP server helper for tool-only agents.
  - Ship a planner-driven multi-agent demo template (A2A + MCP + OTLP) as reference.
  - Add explicit run docs and minimal examples for manual debugging.

## Phase 6: Multi-level memory (Estimate: M)
Goals: short and long-term memory backends.
Dependencies: Phase 0.
Milestone: per-agent memory configuration.
Tasks:
- [x] In-memory backend.
- [x] Persistent backend (file store, vector store).
- [x] Configuration per agent.
- [x] Agent loop reads/writes memory in runtime.
Acceptance: US-06 complete.

## Phase 7: Governance and AGENTS.md (Estimate: M)
Goals: policy enforcement and AGENTS.md loading.
Dependencies: Phase 0, Phase 1.
Milestone: policy and AGENTS.md rules enforced.
Tasks:
- [x] AGENTS.md loader and parser.
- [x] Policy engine (scopes, allow/deny).
- [x] Audit event store.
Follow-ups (post-milestone):
- [x] Config-driven policy rule loading.
- [x] Policy enforcement for A2A/MCP calls (beyond tool gating).
- [x] Server-side policy enforcement for A2A handlers.
- [~] Human-in-the-loop policy flow (approvals + endpoints).
Acceptance: US-07 complete; US-08 in progress.

## Phase 8: UI/CLI (Estimate: L)
Goals: operator visibility and control.
Dependencies: Phase 2, Phase 5.
Milestone: dashboard with agents and traces.
Tasks:
- [ ] CLI for agent status and traces.
- [ ] Web UI for flows, traces, memory state.
Acceptance: US-09 complete.

## Core UX Track (Library + Demo)
Goals: make Kairos approachable to developers from Python agent frameworks while keeping standards.
Dependencies: Phase 5, Phase 7.
Core tasks (library):
- [ ] Role/manifest metadata API (coexists with AgentCard).
- [ ] Task entity in core with traceable IDs/status/result (no proto/store changes).
- [ ] Event taxonomy for semantic streaming/logs (stable types + minimal fields).
Demo tasks:
- [ ] Role YAML files to feed core role metadata (`demoKairos/docs/role-*.yaml`).
- [ ] Narrative guide: “what it is / what it is not”.
- [ ] Demo builder facade (`NewSystem` + `WithAgent` + `WithFlow`), revisit for core after Task/Role/Event stabilize.
- [ ] Single entrypoint script for running demo.
Notes:
- No changes to A2A proto or stores in this track.
- See `docs/walkthrough-demo-improvements.md` for the detailed plan.

## Milestone Dependencies (summary)
- P0 -> P1, P2, P3, P4, P6, P7
- P2 -> P5, P8
- P5 -> P8

## Tracking Updates
Update checkboxes per task and add brief notes under each phase if needed.
See `docs/CONFIGURATION.md` for configuration sources and precedence.
