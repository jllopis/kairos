# Task Core API

Kairos exposes tasks as first-class entities at the library level without
modifying the A2A proto or stores.

## Task model

`pkg/core.Task` provides:

- `ID`, `Goal`, `AssignedTo`
- `Status` (`pending`, `running`, `completed`, `failed`, `cancelled`, `rejected`)
- `Result`, `Error`
- `CreatedAt`, `StartedAt`, `FinishedAt`
- `Metadata`

## Lifecycle helpers

Use helpers to drive task status:

- `Start()`
- `Complete(result)`
- `Fail(error)`
- `Reject(reason)`
- `Cancel(reason)`

## Context propagation

Attach a task to context to propagate `task_id` and `task_goal`:

```go
task := core.NewTask("Summarize sales", "orchestrator")
ctx = core.WithTask(ctx, task)
```

When an agent runs with a task in context, it updates task status and result
automatically.

## Interop with A2A

Task tracking is independent of A2A proto/stores:

- No changes to A2A types or TaskStore.
- `task_id`/`task_goal` are carried in events and logs.
- Streaming events include `event_type` + payload metadata.
