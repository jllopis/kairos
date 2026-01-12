# Walkthrough: Demo A2A Agents (planner + agent + mcp)

This walkthrough describes the demo in `demoKairos/` and how it uses the core Kairos packages to run a multi-agent workflow.

## Goals

- Each agent is built with `pkg/agent` and `pkg/llm`.
- Configuration is loaded via `pkg/config`.
- Telemetry is initialized via `pkg/telemetry` (OTLP if configured).
- MCP tools are exposed and consumed using `pkg/mcp`.
- Agents delegate using `pkg/a2a` with gRPC streaming.
- The orchestrator follows an explicit plan using `pkg/planner`.

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

## Governance policies in the demo

The demo can load governance rules from the same config file used for LLM and telemetry.
Each agent wires the policy engine so it applies to:

- tool calls (agent loop),
- MCP client calls,
- A2A client calls from the orchestrator.

Example config fragment (see `demoKairos/data/demo-config.json`):

```json
{
  "governance": {
    "policies": [
      { "id": "deny-spreadsheet", "effect": "deny", "type": "tool", "name": "query_spreadsheet" },
      { "id": "deny-knowledge", "effect": "deny", "type": "agent", "name": "knowledge-agent" },
      { "id": "deny-mcp", "effect": "deny", "type": "mcp", "name": "spreadsheet-mcp" }
    ]
  }
}
```

Expected behavior when a policy denies a call:

- tool calls: the agent writes a policy observation and skips execution,
- A2A calls: the client returns a permission denied error,
- MCP calls: the client fails fast before sending a request.

## Streaming events

The orchestrator emits semantic progress as `TaskStatusUpdateEvent`:

- `thinking`
- `retrieval.started` / `retrieval.done`
- `tool.started` / `tool.done`
- `response.final`

Incremental text is sent as `StreamResponse_Msg` chunks.

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
