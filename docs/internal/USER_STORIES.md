# User Stories and Acceptance Criteria

For configuration sources and precedence, see `docs/CONFIGURATION.md`.

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
- Document MCP retry/timeout/cache settings in examples/config docs.
Acceptance criteria:
- Create a Go agent via typed API and run a simple task.
- Agent can call an external MCP tool.
- Structured logs are emitted for the run.

## US-02: A2A discovery and delegation
As an architect, I want an agent to delegate tasks to another agent via A2A.
Status: [x] Done
Current implementation (MVP, gRPC binding):
- Types generated from `pkg/a2a/proto/a2a.proto`.
- A2AService server with SendMessage + SendStreamingMessage + Get/List/Cancel Task.
- AgentCard publishing + discovery client.
- Task/Message/Artifact mapping + streaming responses.
- Playbook multi-agent flow (examples/playbook) with orchestrator delegating to knowledge/spreadsheet agents.
Notes:
- HTTP+JSON/JSON-RPC client helpers are available alongside server bindings.
Acceptance criteria:
- Remote agent registers and is discoverable.
- Agent A calls Agent B and receives a response.
- Trace continuity is preserved across agents.

## US-03: Explicit planner with graphs
As an engineer, I want to define deterministic flows using a graph planner.
Status: [x] Done
Current implementation:
- Graph schema with nodes/edges and validation.
- JSON/YAML parsers for graph definitions.
- Executor with per-node tracing, branching conditions, and audit hooks.
- Walkthrough and example usage in `docs/legacy/walkthrough-explicit-planner.md` and `examples/explicit-planner`.
Follow-ups:
- Optional policy evaluation per planner node.
Acceptance criteria:
- Graph defined in YAML/JSON executes correctly.
- Each node is traced with OpenTelemetry spans.
- Graph model can be serialized and deserialized without loss.

## US-04: Emergent planner
As a flow designer, I want the agent to choose the next action dynamically.
Status: [x] Done
Current implementation:
- ReAct-style loop selects tools based on LLM tool calls by default.
- Prompt-driven "Action:" parsing remains as a fallback for providers without tool calls.
- LLM tool-calls are supported (function schema + structured tool calls).
- Tool calls are logged with trace/span identifiers.
- Legacy Action parsing is disabled by default and can be enabled explicitly.
Follow-ups:
- Deprecate legacy Action parsing fully (warn + eventual removal plan).
- Track removal timeline once a release cadence is defined.
Acceptance criteria:
- Agent selects the next step among multiple tools or agents.
- Decisions and intermediate results are logged.

## US-05: End-to-end observability
As an SRE, I want traces, metrics, and logs for multi-agent diagnosis.
Status: [x] Done
Current implementation:
- OpenTelemetry tracer initialization and spans for agent, LLM, tool calls, and memory.
- Basic metrics (run count, errors, latency histograms) via stdout exporter.
- Structured logs include trace/span identifiers.
- Configurable OTLP exporter for traces and metrics.
- Example OTLP config in docs.
- OTLP smoke test and demo validation steps documented.
Follow-ups:
- Add optional automated OTLP validation in CI (behind an env flag).
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
Status: [x] Done
Current implementation:
- AGENTS.md loader that walks upward to find repo instructions.
- Documentation for loader usage in `docs/legacy/walkthrough-governance-agentsmd.md`.
Acceptance criteria:
- AGENTS.md is detected and parsed at startup.
- Rules are applied to agent base context.

## US-08: Governance and policies
As a security owner, I want policies per agent and full auditing.
Status: [~] In progress
Current implementation:
- Policy engine with ordered allow/deny rules and glob matching.
- Config-driven policy rule loading.
- Agent hook to block tool calls using `agent.WithPolicyEngine`.
- A2A and MCP client policy enforcement hooks.
 - Server-side A2A handler policy enforcement with pending approvals.
 - Approval store for HITL workflows (memory/SQLite).
Follow-ups:
- Approval timeouts/expiry policies.
- Operator UI/CLI for approvals and audit views.
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
