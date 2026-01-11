# User Stories and Acceptance Criteria

## Status Legend
- [ ] Planned
- [~] In progress
- [x] Done

## US-01: Run a Go agent with MCP tools
As a developer, I want to instantiate a Go agent with MCP tools to test capabilities quickly.
Status: [x] Done
Current implementation:
- Agent can be created via typed API and run with a mock or Ollama LLM.
- MCP client/server wrappers exist.
- MCP client includes retries, timeouts, and tool discovery caching by default.
- MCP tool adapter supports schema mapping and required-arg validation.
- Agent can auto-discover MCP tools via registered clients, filtered by skills.
- Example MCP agent loads servers from config and executes a tool call.
Follow-ups:
- Expose MCP retry/timeout/cache policy in config and docs.
- Add end-to-end MCP smoke tests for stdio + HTTP.
Acceptance criteria:
- Create a Go agent via typed API and run a simple task.
- Agent can call an external MCP tool.
- Structured logs are emitted for the run.

## US-02: A2A discovery and delegation
As an architect, I want an agent to delegate tasks to another agent via A2A.
Status: [ ] Planned
Implementation plan (MVP, gRPC binding):
- Types generated from `docs/protocols/A2A/a2a.proto`.
- A2AService server with SendMessage + SendStreamingMessage + Get/List/Cancel Task.
- AgentCard publishing + discovery client.
- Task/Message/Artifact mapping with trace propagation.
Gaps to close:
- AuthN/AuthZ middleware (OIDC/mTLS) and multi-tenant routing.
- Conformance tests across bindings (JSON-RPC/HTTP+JSON).
Acceptance criteria:
- Remote agent registers and is discoverable.
- Agent A calls Agent B and receives a response.
- Trace continuity is preserved across agents.

## US-03: Explicit planner with graphs
As an engineer, I want to define deterministic flows using a graph planner.
Status: [~] In progress
Current implementation:
- Graph schema with nodes/edges and validation.
- JSON/YAML parsers for graph definitions.
- Executor with per-node tracing, branching conditions, and audit hooks.
- Walkthrough and example usage in `docs/walkthrough-explicit-planner.md` and `examples/explicit-planner`.
Gaps to close:
- Richer audit events (persisted store) and advanced condition types.
Acceptance criteria:
- Graph defined in YAML/JSON executes correctly.
- Each node is traced with OpenTelemetry spans.
- Graph model can be serialized and deserialized without loss.

## US-04: Emergent planner
As a flow designer, I want the agent to choose the next action dynamically.
Status: [~] In progress
Current implementation:
- ReAct-style loop selects tools based on LLM tool calls by default.
- Prompt-driven "Action:" parsing remains as a fallback for providers without tool calls.
- LLM tool-calls are supported (function schema + structured tool calls).
- Tool calls are logged with trace/span identifiers.
 - Legacy Action parsing is disabled by default and can be enabled explicitly.
Gaps to close:
- Prefer tool calls over string parsing when supported (deprecate "Action:" path).
- Deprecate legacy Action parsing (warn + eventual removal plan).
 - Document fallback deprecation phases and activation rules.
Acceptance criteria:
- Agent selects the next step among multiple tools or agents.
- Decisions and intermediate results are logged.

## US-05: End-to-end observability
As an SRE, I want traces, metrics, and logs for multi-agent diagnosis.
Status: [~] In progress
Current implementation:
- OpenTelemetry tracer initialization and spans for agent, LLM, tool calls, and memory.
- Basic metrics (run count, errors, latency histograms) via stdout exporter.
- Structured logs include trace/span identifiers.
- Configurable OTLP exporter for traces and metrics.
- Example OTLP config in docs.
Gaps to close:
- Validate OTLP export against a backend and document verification steps.
Acceptance criteria:
- Traces exported to a standard OTel backend.
- Basic metrics (latency, errors) exported.
- Logs include trace/span identifiers.

## US-06: Multi-level memory
As a user, I want short and long-term memory for agents.
Status: [x] Done
Current implementation:
- Memory interface with in-memory, file-backed, and vector (Qdrant) backends.
- Per-agent memory attachment via agent options.
Acceptance criteria:
- Memory interface supports Store/Retrieve.
- At least one in-memory and one persistent implementation exists.
- Memory can be configured per agent.

## US-07: AGENTS.md auto-loading
As an operator, I want AGENTS.md to load automatically on startup.
Status: [ ] Planned
Acceptance criteria:
- AGENTS.md is detected and parsed at startup.
- Rules are applied to agent base context.

## US-08: Governance and policies
As a security owner, I want policies per agent and full auditing.
Status: [ ] Planned
Acceptance criteria:
- Scopes can be defined per tool/skill.
- All executions are audited with metadata.
- Human-in-the-loop can be enabled.

## US-09: Control UI
As an operator, I want a UI to inspect agents and traces.
Status: [ ] Planned
Acceptance criteria:
- Dashboard shows agents, flows, and traces.
- Memory and step state can be inspected.
