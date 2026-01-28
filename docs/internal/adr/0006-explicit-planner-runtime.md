# 6. Explicit Planner Integration in Agent Runtime

Date: 2026-01-28

## Status

Accepted

## Context

Kairos includes an explicit planner (`pkg/planner`) with a graph executor, but
the runtime agent loop only supported the emergent ReAct pattern. This created
two separate execution models and forced users to implement planner wiring
manually for deterministic workflows.

We need a unified runtime where a single agent can execute:
- **Emergent (ReAct)** reasoning when no plan is provided.
- **Explicit (planner graph)** when deterministic orchestration is required.

## Decision

### 1) Planner as an Agent Mode

If an agent is configured with a planner graph, `agent.Run(...)` executes the
explicit planner. Otherwise, it runs the emergent ReAct loop.

**New APIs:**
- `agent.WithPlanner(graph)`
- `agent.WithPlannerHandlers(handlers)`
- `agent.WithPlannerAuditStore(store)`
- `agent.WithPlannerAuditHook(hook)`

### 2) Default Node Types in Runtime

The runtime provides handlers for common node types:

- `tool`: execute `node.tool` or `metadata.tool`.
- `agent`: run the emergent loop inside the node.
- `llm`: direct LLM call without tool calling.
- `decision` / `noop`: no-op, preserving `state.Last`.
- If `node.type` matches a tool name, the tool is executed.

### 3) State and Inputs

- If a node has no `input`, the handler uses `state.Last`.
- The initial user input is available as `state.Outputs["input"]`.
- If memory is configured, the resolved context is exposed as `state.Outputs["memory"]`.

### 4) Observability + Audit

Planner execution emits:
- OTEL spans per node (`Planner.Node`).
- Optional audit events through `planner.AuditStore` and `AuditHook`.

## Consequences

### Positive
- Deterministic workflows are first-class in the runtime.
- A single agent API supports both explicit and emergent planning.
- Observability is consistent across planners and agent execution.

### Negative
- More runtime branching in `agent.Run`.
- Default node type semantics must be documented to avoid confusion.

## Follow-ups

- Extend docs/API and planner docs with node type contracts.
- Add examples using `kairos run --plan`.
