# Roadmap and Phases

## Status Legend
- [ ] Planned
- [~] In progress
- [x] Done

## Milestones
- M0: Go SDK skeleton + hello agent (Phase 0) [x]
- M1: Agent can call external MCP tool (Phase 1) [~]
- M2: OTel traces visible in backend (Phase 2) [~]
- M3: YAML/JSON graph executes end-to-end (Phase 3) [ ]
- M4: Emergent flow runs with decision logs (Phase 4) [~]
- M5: Two agents delegate with distributed traces (Phase 5) [ ]
- M6: Per-agent memory with short/long backends (Phase 6) [x]
- M7: AGENTS.md and policies enforced with audit trail (Phase 7) [ ]
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
- [~] Skill -> MCP tool binding (Agent tool adapter).
- [x] Tool schema/arguments mapping and validation.
- [ ] Error handling and retries.
Acceptance: US-01 complete.

## Phase 2: Observability baseline (Estimate: M)
Goals: OTel traces/metrics/logs from runtime.
Dependencies: Phase 0.
Milestone: traces visible in OTel backend.
Tasks:
- [x] OTel tracer and span propagation in runtime (Agent/Runtime + tool/memory/LLM spans).
- [x] Metrics for latency and error counts.
- [x] Structured logs with trace/span ids.
Acceptance: US-05 (partial).

## Phase 3: Explicit planner (Estimate: L)
Goals: deterministic graph execution.
Dependencies: Phase 0, Phase 1.
Milestone: YAML/JSON graph executes end-to-end.
Tasks:
- [ ] Graph model and executor.
- [ ] YAML/JSON parser and serializer.
- [ ] Per-node tracing and audit events.
Acceptance: US-03 complete.

## Phase 4: Emergent planner (Estimate: M)
Goals: dynamic next-step decisions.
Dependencies: Phase 0, Phase 1.
Milestone: emergent flow runs with decision logs.
Tasks:
- [~] Decision engine with tool selection (ReAct loop).
- [ ] Logging of decisions and outcomes.
- [ ] Structured tool-call parsing (avoid brittle string parsing).
Acceptance: US-04 complete.

## Phase 5: A2A distributed runtime (Estimate: L)
Goals: discovery, delegation, and trace continuity.
Dependencies: Phase 0, Phase 2.
Milestone: two agents delegate with distributed traces.
Tasks:
- [ ] A2A discovery and registration.
- [ ] Remote agent invocation (call/response).
- [ ] Trace context propagation over A2A.
Acceptance: US-02 complete, US-05 complete.

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
- [ ] AGENTS.md loader and parser.
- [ ] Policy engine (scopes, allow/deny).
- [ ] Audit event store.
Acceptance: US-07, US-08 complete.

## Phase 8: UI/CLI (Estimate: L)
Goals: operator visibility and control.
Dependencies: Phase 2, Phase 5.
Milestone: dashboard with agents and traces.
Tasks:
- [ ] CLI for agent status and traces.
- [ ] Web UI for flows, traces, memory state.
Acceptance: US-09 complete.

## Milestone Dependencies (summary)
- P0 -> P1, P2, P3, P4, P6, P7
- P2 -> P5, P8
- P5 -> P8

## Tracking Updates
Update checkboxes per task and add brief notes under each phase if needed.
