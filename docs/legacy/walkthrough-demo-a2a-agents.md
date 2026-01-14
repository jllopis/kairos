# Walkthrough: Demo A2A Agents (planner + agent + mcp)

This walkthrough describes the demo in `demoKairos/` and how it uses the core Kairos packages to run a multi-agent workflow.

## Goals

- Each agent is built with `pkg/agent` and `pkg/llm`.
- Configuration is loaded via `pkg/config`.
- Telemetry is initialized via `pkg/telemetry` (OTLP if configured).
- MCP tools are exposed and consumed using `pkg/mcp`.
- Agents delegate using `pkg/a2a` with gRPC streaming.
- The orchestrator follows an explicit plan using `pkg/planner`.
- Role manifests are attached via `pkg/core` metadata for clarity.
- Tasks are tracked as first-class entities via `pkg/core.Task` (no proto changes).

## Demo structure

- `demoKairos/cmd/orchestrator`: planner-driven orchestrator + streaming A2A server.
- `demoKairos/cmd/knowledge`: knowledge agent with MCP tool `retrieve_domain_knowledge` (Qdrant-backed RAG).
- `demoKairos/cmd/spreadsheet`: spreadsheet agent with MCP tools (`query_spreadsheet`, `list_sheets`, `get_schema`).
- `demoKairos/cmd/client`: gRPC streaming client for the demo.
- `demoKairos/data/orchestrator_plan.yaml`: workflow plan.

## Planner (orchestrator_plan.yaml)

```yaml
id: demo-orchestrator
start: detect_intent
nodes:
  detect_intent:
    id: detect_intent
    type: detect_intent
  knowledge:
    id: knowledge
    type: knowledge
  spreadsheet:
    id: spreadsheet
    type: spreadsheet
  synthesize:
    id: synthesize
    type: synthesize
edges:
  - from: detect_intent
    to: knowledge
  - from: knowledge
    to: spreadsheet
  - from: spreadsheet
    to: synthesize
```

Each node is mapped to a handler in the orchestrator:

- `detect_intent`: LLM classifier (pkg/agent)
- `knowledge`: A2A call to knowledge agent
- `spreadsheet`: A2A call to spreadsheet agent
- `synthesize`: LLM synthesis (pkg/agent)

## MCP usage

- Knowledge agent exposes `retrieve_domain_knowledge` via MCP.
- Spreadsheet agent exposes `query_spreadsheet`, `list_sheets`, `get_schema` via MCP.
- Each agent uses `agent.WithMCPClients(...)` to call its local MCP server.

## Streaming events

The orchestrator emits semantic progress as `TaskStatusUpdateEvent` using the
core event taxonomy (`docs/EVENT_TAXONOMY.md`):

- `agent.task.started`
- `agent.thinking`
- `agent.delegation`
- `agent.task.completed`
- `agent.error`

Each event includes `event_type` + `payload` in metadata. Incremental text is
sent as `StreamResponse_Msg` chunks.

## Role manifests

Each demo agent loads a role manifest from `demoKairos/docs/role-*.yaml` and
attaches it via `agent.WithRoleManifest`. This is a library-level API for
semantic role metadata that complements AgentCard.

## Task tracking (core)

The orchestrator creates a `core.Task` and attaches it to context with
`core.WithTask`. This enables:

- `task_id` and `task_goal` propagation in events/logs.
- A stable task model without touching A2A proto or stores.

## Runtime flow

1. Client calls `SendStreamingMessage` on the orchestrator.
2. Orchestrator runs the planner graph and streams status updates.
3. Knowledge agent handles RAG via MCP tool + Qdrant.
4. Spreadsheet agent queries CSV via MCP tool.
5. Orchestrator synthesizes the final response and streams it.

## Suggested extensions

- Add SSE gateway for HTTP streaming.
- Add an MCP tool to the orchestrator for auditing or persistence.
- Add AgentCard discovery checks before delegation.
