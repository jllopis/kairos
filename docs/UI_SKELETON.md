# UI Skeleton (Phase 8.3)

Este documento define el esqueleto inicial de la UI web para Kairos (operador).
No introduce nuevos endpoints; reutiliza A2A + approvals + AgentCard.

## Objetivo

- Pantallas minimas para operar y observar agentes, tasks, approvals y trazas.
- UI neutra que se puede montar sobre el CLI/HTTP existente.

## Pantallas

### 1) Agents
- Lista de agentes con nombre, descripcion, version, endpoint.
- Fuente: AgentCard discovery (`/.well-known/agent-card.json`).
- Filtro basico por nombre.

### 2) Tasks
- Tabla con `task_id`, estado, updated_at, mensaje.
- Detalle de task con historial y artifacts.
- Fuente: A2A `ListTasks` + `GetTask`.

### 3) Task Stream / Trace
- Panel de streaming con eventos semanticos.
- Fuente: A2A `SubscribeToTask`.
- Mostrar `event_type`, `trace_id`, mensaje, payload.

### 4) Approvals
- Tabla de approvals pendientes y filtrado por estado.
- Acciones aprobar / rechazar.
- Fuente: A2A HTTP+JSON approvals.

## Endpoints requeridos (ya existen)

- AgentCard discovery:
  - `GET /.well-known/agent-card.json`
- Tasks:
  - `ListTasks`, `GetTask`, `SubscribeToTask`, `CancelTask`
- Approvals:
  - `GET /approvals`, `GET /approvals/{id}`
  - `POST /approvals/{id}:approve`, `POST /approvals/{id}:reject`

## UX minima

- Routing: `/agents`, `/tasks`, `/tasks/:id`, `/approvals`.
- Tabla + panel de detalle (split view).
- Streaming con autoscroll y pausa.

## No incluye

- Persistencia de UI.
- Autenticacion/autorizacion.
- Multi-tenant.

