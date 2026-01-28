# 7. Planner Handler Resolution by Node ID (Opt-in)

Date: 2026-01-28

## Status

Accepted

## Context

The explicit planner resolves node handlers by `node.type`, which is stable and
portable across environments. Some workflows need per-node overrides (tests,
hotfixes, tenant-specific behavior) without introducing many new types or
modifying the graph definition.

We need an opt-in mechanism to target individual nodes without making `node.id`
first-class or required for normal execution.

## Decision

- Keep `node.type` as the primary and default handler resolution key.
- Add an **opt-in** override map by `node.id` with higher precedence:
  - `planner.Executor.HandlersByID`
  - `agent.WithPlannerIDHandlers(...)`
- Resolution order: **id override → type handler**.

## Consequences

### Positive
- Enables per-node overrides without modifying plan YAML/JSON.
- Supports testing, hotfixes, and environment-specific behavior.
- Keeps the core contract portable and centered on `node.type`.

### Negative
- `node.id` becomes a potential implicit dependency for overrides.
- Renaming nodes can silently break overrides if not tracked.

## Follow-ups

- Document handler resolution order in planner docs and API reference.
