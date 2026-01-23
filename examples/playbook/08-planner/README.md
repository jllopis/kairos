# Playbook 08 - Planner (Explicit Graph)

Goal: execute a deterministic plan with audit logging.

## Why this step?

While LLMs are great at reasoning, some processes (like processing a payment or booking a flight) must be **deterministic**. You don't want an LLM "hallucinating" the booking sequence. The Planner allows you to define a strict state machine or graph that the AI follows.

## SkyGuide Narrative

SkyGuide is now ready to book flights. To ensure we don't charge the user before confirming the seat is available, we use an explicit Plan. We also need an "audit trail" to see exactly why a booking succeeded or failed.

## Incremental reuse

- Add `internal/planner` for graph loading and executor wiring.

## What to implement

- Define a flight booking graph in JSON or YAML (Availability -> Payment -> Confirmation).
- Load it with `planner.ParseJSON` or `planner.ParseYAML`.
- Create handlers for node types (agent/tool/decision).
- Execute with `planner.Executor`.
- Persist audit events with `planner.NewSQLiteAuditStore`.
- Reuse provider/config wiring from step 02 via shared helpers.

## Suggested checks

- Graph execution follows the expected path.
- Audit events are written to the SQLite database and can be queried.

## Manual tests

- Execute the booking graph once with a flight request payload.

## Expected behavior

- Each node runs in the defined order and produces output.
- The audit store records the start and finish timestamp for each node.

## Checklist

- [ ] Graph validates before execution.
- [ ] Audit rows are persisted in the database.

## References

- [06-explicit-planner](file:///Users/jllopis/src/kairos/examples/06-explicit-planner)
- `pkg/planner`
