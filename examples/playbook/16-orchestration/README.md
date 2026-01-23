# Playbook 15 - Orchestration (Final Goal)

Goal: multi-agent system with webhook input, DB lookup, enrichment, and formatting.

Incremental reuse:
- Add `internal/http` for the webhook, `internal/a2a` for agent wiring,
  and `internal/store` for DB access and connectors.

Roles and responsibilities:
- Orchestrator: validate request, select plan, delegate to specialists, merge results.
- DB specialist: SQLConnector read-only access, returns records + provenance.
- Enricher A: external API enrichment (OpenAPI or MCP).
- Enricher B: knowledge/context enrichment (GraphQL or MCP).
- Formatter: produces final response in requested format and language.

Webhook request fields (suggested):
- request_id: unique id for tracing and idempotency.
- session_id: used for conversation memory.
- user_id: for access policies and personalization.
- instruction: the user prompt.
- locale: language for the response.
- response_format: short_text, json, or markdown.
- urgency: low, normal, high (used by planner).
- max_sources: limit for enrichment fan-out.

Webhook response fields (suggested):
- request_id, session_id, status.
- answer: final formatted response.
- sources: list of data sources used (db, openapi, graphql, mcp).
- warnings: guardrail or policy warnings (if any).
- trace_id: for debugging and observability.

SQL schema (minimum data model):
- customers: id, name, email, status, created_at.
- orders: id, customer_id, total, currency, created_at.
- tickets: id, customer_id, topic, status, created_at.
Indexes and constraints (guidance):
- customers.email unique.
- orders.customer_id indexed.
- tickets.customer_id indexed.
- status fields should be enumerated.

Target flow:
1) Webhook receives user instruction.
2) Guardrails check input.
3) Orchestrator agent delegates to specialists.
4) DB agent reads from SQL connector (read-only).
5) Enricher agents add context (OpenAPI, MCP, or GraphQL).
6) Formatter agent produces the final response.
7) Guardrails filter output and return to caller.

What to implement:
- Agents:
  - Orchestrator
  - DB specialist
  - Enricher(s)
  - Formatter
- A2A servers for each agent (`pkg/a2a/server`).
- AgentCards and discovery (`pkg/a2a/agentcard`, `pkg/discovery`).
- Webhook HTTP server that calls the orchestrator.
- Use `core.Task` + `core.WithSessionID` per request.
- Policies + tool filters to restrict each agent.
- Telemetry spans around A2A hops and DB calls.
- Reuse provider/config wiring from step 02 via shared helpers.
- Keep provider selection limited to mock, ollama, openai, gemini.

Suggested checks:
- Start each agent in its own process.
- `curl` the webhook and get a formatted response.
- Logs show A2A hops and tool calls.

Manual tests:
- "Given customer email X, summarize their latest order and open tickets."

Expected behavior:
- DB agent returns customer, order, and ticket data.
- Enrichers add context if needed.
- Formatter returns a clean summary.

Checklist:
- [ ] Orchestrator uses A2A for delegation.
- [ ] DB agent uses SQLConnector in read-only mode.
- [ ] Guardrails applied to input and output.
- [ ] Response includes sources and warnings.

References:
- `pkg/a2a`
- `pkg/discovery`
- `pkg/connectors/sql.go`
- `examples/07-multi-agent-mcp`
