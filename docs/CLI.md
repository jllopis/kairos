# Kairos CLI (Fase 8.1 MVP)

Este documento define la interfaz del CLI MVP y los endpoints base que consume.

## Objetivos

- Operación básica: status, agents, tasks, aprobaciones, MCP tools.
- Salida humana por defecto con opción JSON (`--json`).
- Reutilizar APIs existentes sin tocar proto ni stores.

## Flags globales

- `--config` ruta a `settings.json` (mismo cargador que runtime).
- `--set key=value` overrides (igual que `config.LoadWithCLI`).
- `--grpc` dirección A2A gRPC (por defecto: `localhost:8080`).
- `--http` base URL A2A HTTP+JSON (por defecto: `http://localhost:8080`).
- `--json` salida JSON.
- `--timeout` timeout de llamadas (por defecto: `30s`).
- `--web` inicia la UI web mínima (HTMX).
- `--web-addr` dirección de bind para la UI (por defecto `:8088`).

Variables de entorno sugeridas:
- `KAIROS_GRPC_ADDR`
- `KAIROS_HTTP_URL`
- `KAIROS_AGENT_CARD_URLS` (lista separada por comas)

## Comandos MVP

### `kairos status`
Muestra versión del CLI, endpoints configurados y resultado de healthcheck básico.

### `kairos agents list`
Descubre AgentCards desde URLs provistas por `--agent-card` (repeatable) o
`KAIROS_AGENT_CARD_URLS`. La salida incluye nombre, endpoint A2A, capacidades y
metadata.

### `kairos tasks list`
Filtros: `--status`, `--context`, `--page-size`, `--page-token`.
Salida: id, estado, updated_at, resumen.

### `kairos tasks follow <task_id>`
Sigue `TaskStatusUpdateEvent` y streaming semántico. Formatea con `EventType`
(ver `docs/EVENT_TAXONOMY.md`). `--out <path>` escribe JSON lines del stream.

### `kairos approvals list`
Filtros: `--status`, `--expires-before`.
Salida: id, status, reason, created_at, expires_at.

### `kairos approvals approve|reject <id>`
`--reason` para justificación.

### `kairos mcp list`
Lee `mcp.servers` desde config y lista tools por servidor. La salida incluye
nombre/URL del servidor y tools (name/description/input schema).

## UI mínima (Fase 8.3)

Ejecuta la interfaz web con:

```
kairos --web --config docs/internal/demo-settings.json
```

Opcional:

```
kairos --web --web-addr :8090 --config docs/internal/demo-settings.json
```

La UI usa HTMX y reutiliza los endpoints A2A y aprobaciones. La carga de HTMX usa
CDN por defecto; se puede servir localmente más adelante.

## Comandos Fase 8.2

### `kairos tasks cancel <task_id>`
Cancela un task vía `CancelTask`.

### `kairos tasks retry <task_id>`
Reintenta enviando el último mensaje `USER` del task como una nueva solicitud.
`--history-length` controla cuánto historial se inspecciona (por defecto: 50).

### `kairos traces tail --task <task_id>`
Sigue el stream de un task y muestra `event_type` y `trace_id` si están
presentes. `--out <path>` escribe JSON lines del stream.

### `kairos approvals tail`
Polling periódico de aprobaciones para ver nuevas entradas.
Flags: `--status` (por defecto: `pending`), `--interval` (por defecto: `5s`), `--out`.

### `kairos registry serve`
Arranca un registry HTTP mínimo con TTL.
Flags: `--addr` (por defecto: `:9900`), `--ttl` (por defecto: `30s`).

## Mapeo a endpoints

- A2A gRPC:
  - `ListTasks`, `GetTask`, `SubscribeToTask`.
- A2A HTTP+JSON:
  - `GET /approvals`, `GET /approvals/{id}`.
  - `POST /approvals/{id}:approve`, `POST /approvals/{id}:reject`.
- AgentCard discovery:
  - `GET /.well-known/agent-card.json` por cada URL configurada.
- MCP:
  - `mcp.servers[*]` desde config, usando `pkg/mcp` para `ListTools`.

## Notas

- Las aprobaciones no están en el proto A2A; el CLI usa HTTP+JSON para estos endpoints.
- `tasks follow` usa gRPC streaming; si el servidor no soporta `SubscribeToTask`, se devuelve error claro.
