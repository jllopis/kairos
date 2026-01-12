# Kairos CLI (Phase 8.1 MVP)

Este documento define la interfaz del CLI MVP y los endpoints base que consume.

## Objetivos

- Operacion basica: status, agents, tasks, approvals, MCP tools.
- Salida humana por defecto con opcion JSON (`--json`).
- Reutilizar APIs existentes sin tocar proto ni stores.

## Flags globales

- `--config` path a `settings.json` (mismo loader que runtime).
- `--set key=value` overrides (igual que `config.LoadWithCLI`).
- `--grpc` direccion A2A gRPC (default: `localhost:8080`).
- `--http` base URL A2A HTTP+JSON (default: `http://localhost:8080`).
- `--json` salida JSON.
- `--timeout` timeout de llamadas (default: `30s`).
- `--web` inicia la UI web minima (HTMX).
- `--web-addr` bind address para la UI (default `:8088`).

Variables de entorno sugeridas:
- `KAIROS_GRPC_ADDR`
- `KAIROS_HTTP_URL`
- `KAIROS_AGENT_CARD_URLS` (lista separada por comas)

## Comandos MVP

### `kairos status`
- Muestra version del CLI, endpoints configurados y resultado de healthcheck basico.

### `kairos agents list`
- Descubre AgentCards desde URLs provistas por `--agent-card` (repeatable) o `KAIROS_AGENT_CARD_URLS`.
- Salida: nombre, endpoint A2A, capacidades, metadata.

### `kairos tasks list`
- Filtros: `--status`, `--context`, `--page-size`, `--page-token`.
- Salida: id, estado, updated_at, resumen.

### `kairos tasks follow <task_id>`
- Sigue `TaskStatusUpdateEvent` y streaming semantico.
- Formatea con `EventType` (ver `docs/EVENT_TAXONOMY.md`).
- `--out <path>` escribe JSON lines del stream.

### `kairos approvals list`
- Filtros: `--status`, `--expires-before`.
- Salida: id, status, reason, created_at, expires_at.

### `kairos approvals approve|reject <id>`
- `--reason` para justificacion.

### `kairos mcp list`
- Lee `mcp.servers` desde config y lista tools por servidor.
- Salida: server name/url, tools (name/description/input schema).

## UI minima (Phase 8.3)

Ejecuta la interfaz web con:

```
kairos --web
```

Opcional:

```
kairos --web --web-addr :8090
```

La UI usa HTMX y reutiliza los endpoints A2A y approvals.
Nota: la carga de HTMX usa CDN por defecto; se puede servir localmente mas adelante.

## Comandos Phase 8.2

### `kairos tasks cancel <task_id>`
- Cancela un task via `CancelTask`.

### `kairos tasks retry <task_id>`
- Reintenta enviando el ultimo mensaje `USER` del task como una nueva solicitud.
- `--history-length` controla cuanto historial se inspecciona (default: 50).

### `kairos traces tail --task <task_id>`
- Sigue el stream de un task y muestra `event_type` y `trace_id` si estan presentes.
- `--out <path>` escribe JSON lines del stream.

### `kairos approvals tail`
- Polling periodico de approvals para ver nuevas entradas.
- Flags: `--status` (default: `pending`), `--interval` (default: `5s`), `--out`.

### `kairos registry serve`
- Arranca un registry HTTP minimo con TTL.
- Flags: `--addr` (default: `:9900`), `--ttl` (default: `30s`).

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

- Approvals no estan en el proto A2A; el CLI usa HTTP+JSON para estos endpoints.
- `tasks follow` usa gRPC streaming; si el server no soporta `SubscribeToTask`, se devuelve error claro.
