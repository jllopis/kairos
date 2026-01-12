# Event Taxonomy

Kairos emits semantic events to make streaming and logs human-friendly and
UI-ready. These events are stable, minimal, and extensible via payloads.

## Event types (stable)

- `agent.thinking`
- `agent.task.started`
- `agent.task.completed`
- `agent.delegation`
- `agent.error`

## Minimal fields

Each event carries the following fields:

```json
{
  "type": "agent.task.started",
  "agent": "orchestrator",
  "task_id": "task-123",
  "timestamp": "2025-01-01T12:00:00Z",
  "payload": {
    "run_id": "run-abc",
    "stage": "detect_intent",
    "task_goal": "Detect user intent"
  }
}
```

## Streaming representation (A2A)

For A2A streaming, events are encoded in the `TaskStatusUpdateEvent.Metadata`
as:

```json
{
  "event_type": "agent.task.started",
  "payload": { "run_id": "run-abc" }
}
```

The message itself still carries the agent text, and the task state tracks the
runtime lifecycle.

## Usage guidelines

- Use **event type** for stable semantics.
- Use **payload** for details such as `stage`, `target`, `tool`, or `intent`.
- Avoid adding new event types unless absolutely necessary.
