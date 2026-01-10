# User Stories and Acceptance Criteria

## US-01: Run a Go agent with MCP tools
As a developer, I want to instantiate a Go agent with MCP tools to test capabilities quickly.
Acceptance criteria:
- Create a Go agent via typed API and run a simple task.
- Agent can call an external MCP tool.
- Structured logs are emitted for the run.

## US-02: A2A discovery and delegation
As an architect, I want an agent to delegate tasks to another agent via A2A.
Acceptance criteria:
- Remote agent registers and is discoverable.
- Agent A calls Agent B and receives a response.
- Trace continuity is preserved across agents.

## US-03: Explicit planner with graphs
As an engineer, I want to define deterministic flows using a graph planner.
Acceptance criteria:
- Graph defined in YAML/JSON executes correctly.
- Each node is traced with OpenTelemetry spans.
- Graph model can be serialized and deserialized without loss.

## US-04: Emergent planner
As a flow designer, I want the agent to choose the next action dynamically.
Acceptance criteria:
- Agent selects the next step among multiple tools or agents.
- Decisions and intermediate results are logged.

## US-05: End-to-end observability
As an SRE, I want traces, metrics, and logs for multi-agent diagnosis.
Acceptance criteria:
- Traces exported to a standard OTel backend.
- Basic metrics (latency, errors) exported.
- Logs include trace/span identifiers.

## US-06: Multi-level memory
As a user, I want short and long-term memory for agents.
Acceptance criteria:
- Memory interface supports Store/Retrieve.
- At least one in-memory and one persistent implementation exists.
- Memory can be configured per agent.

## US-07: AGENTS.md auto-loading
As an operator, I want AGENTS.md to load automatically on startup.
Acceptance criteria:
- AGENTS.md is detected and parsed at startup.
- Rules are applied to agent base context.

## US-08: Governance and policies
As a security owner, I want policies per agent and full auditing.
Acceptance criteria:
- Scopes can be defined per tool/skill.
- All executions are audited with metadata.
- Human-in-the-loop can be enabled.

## US-09: Control UI
As an operator, I want a UI to inspect agents and traces.
Acceptance criteria:
- Dashboard shows agents, flows, and traces.
- Memory and step state can be inspected.
