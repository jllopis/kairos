# Task (API core)

En Kairos, Task es una entidad de primer nivel en la librería sin tocar el
proto A2A ni los stores.

## Modelo de Task

`pkg/core.Task` expone:

- `ID`, `Goal`, `AssignedTo`
- `Status` (`pending`, `running`, `completed`, `failed`, `cancelled`, `rejected`)
- `Result`, `Error`
- `CreatedAt`, `StartedAt`, `FinishedAt`
- `Metadata`

## Ciclo de vida

Helpers disponibles:

- `Start()`
- `Complete(result)`
- `Fail(error)`
- `Reject(reason)`
- `Cancel(reason)`

## Contexto y propagación

Puedes adjuntar una Task al contexto para propagar `task_id` y `task_goal`:

```go
task := core.NewTask("Resumir ventas", "orchestrator")
ctx = core.WithTask(ctx, task)
```

Cuando un agente corre con una Task en contexto, actualiza estado y resultado
automáticamente.

## Interop con A2A

El tracking de Task es independiente del proto/store A2A:

- No hay cambios en tipos A2A ni en TaskStore.
- `task_id`/`task_goal` viajan en eventos y logs.
- Los eventos de streaming incluyen `event_type` y metadata de payload.
