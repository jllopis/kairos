# Taxonomía de eventos

Kairos emite eventos semánticos para que el streaming y los logs sean legibles
por humanos y fáciles de consumir por una UI. Son eventos estables, mínimos y
ampliables vía `payload`.

## Tipos de evento (estables)

Los tipos base son cinco y no deberían crecer sin necesidad. Úsalos como un
vocabulario común y mete el detalle en `payload`.

Tipos disponibles:

- `agent.thinking`
- `agent.task.started`
- `agent.task.completed`
- `agent.delegation`
- `agent.error`

## Campos mínimos

Cada evento incluye estos campos:

```json
{
  "type": "agent.task.started",
  "agent": "orchestrator",
  "task_id": "task-123",
  "timestamp": "2025-01-01T12:00:00Z",
  "payload": {
    "run_id": "run-abc",
    "stage": "detect_intent",
    "task_goal": "Detectar intención de usuario"
  }
}
```

## Representación en streaming (A2A)

En A2A, los eventos se envían dentro de `TaskStatusUpdateEvent.Metadata`:

```json
{
  "event_type": "agent.task.started",
  "payload": { "run_id": "run-abc" }
}
```

El mensaje sigue llevando el texto del agente y el estado de la tarea refleja
el ciclo de vida del runtime.

## Recomendaciones de uso

Usa `type` para la semántica estable y `payload` para detalles como `stage`,
`target`, `tool` o `intent`. Evita crear tipos nuevos salvo que sea necesario.

Si necesitas más semántica, define un campo nuevo en `payload` antes de
inventar otro `type`.
