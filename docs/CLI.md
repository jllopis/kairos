# Kairos CLI

Este documento define la interfaz del CLI de Kairos y los comandos disponibles.

## Objetivos

- Scaffolding de proyectos con `kairos init`
- Operación básica: status, agents, tasks, aprobaciones, MCP tools
- Salida humana por defecto con opción JSON (`--json`)

## Flags globales

- `--config` ruta a `settings.json` (mismo cargador que runtime)
- `--set key=value` overrides (igual que `config.LoadWithCLI`)
- `--grpc` dirección A2A gRPC (por defecto: `localhost:8080`)
- `--http` base URL A2A HTTP+JSON (por defecto: `http://localhost:8080`)
- `--json` salida JSON
- `--timeout` timeout de llamadas (por defecto: `30s`)
- `--web` inicia la UI web mínima (HTMX)
- `--web-addr` dirección de bind para la UI (por defecto `:8088`)

Variables de entorno sugeridas:
- `KAIROS_GRPC_ADDR`
- `KAIROS_HTTP_URL`
- `KAIROS_AGENT_CARD_URLS` (lista separada por comas)

---

## Comandos de Scaffolding

### `kairos init <directory>`

Genera un nuevo proyecto Kairos con estructura recomendada.

**Flags:**
- `-module <path>` (requerido): Go module path (ej: `github.com/myorg/my-agent`)
- `-type <archetype>`: Tipo de proyecto (default: `assistant`)
  - `assistant`: Agente básico con memoria y conversación
  - `tool-agent`: Agente con tools locales y MCP
  - `coordinator`: Coordinador multi-agente con planner
  - `policy-heavy`: Agente con governance estricto
- `-llm <provider>`: Proveedor LLM (default: `ollama`). Opciones: `ollama`, `mock`
- `-mcp`: Incluir configuración MCP
- `-a2a`: Incluir endpoint A2A
- `-corporate`: Incluir infraestructura enterprise (CI/CD, Docker, observabilidad)
- `-overwrite`: Sobreescribir archivos existentes

**Ejemplos:**
```bash
# Proyecto básico
kairos init -module github.com/myorg/my-agent my-agent

# Con tools MCP
kairos init -module github.com/myorg/my-agent -type tool-agent -mcp my-agent

# Coordinador multi-agente
kairos init -module github.com/myorg/my-agent -type coordinator -a2a my-agent

# Proyecto enterprise con CI/CD y observabilidad
kairos init -module github.com/myorg/my-agent -corporate my-agent
```

**Estructura generada:**
```
my-agent/
├── cmd/agent/main.go           # Entrypoint
├── internal/
│   ├── app/app.go              # Wiring de componentes
│   ├── config/config.go        # Loader de configuración
│   └── observability/otel.go   # Setup OTEL
├── config/
│   ├── config.yaml             # Config base
│   ├── config.dev.yaml         # Override desarrollo
│   └── config.prod.yaml        # Override producción
├── Makefile
├── go.mod
└── README.md
```

---

## Comandos de Validación

### `kairos validate`

Valida la configuración actual y verifica conectividad con servicios externos.

**Verificaciones:**
- ✓ Config: Carga correcta del archivo de configuración
- ✓ LLM: Conectividad con el proveedor (Ollama, OpenAI, etc.)
- ✓ MCP: Servidores HTTP alcanzables y con tools disponibles
- ✓ Governance: Políticas bien formadas
- ✓ Skills: Directorios de skills válidos

**Ejemplos:**
```bash
# Validación básica
kairos validate

# Salida JSON (útil para CI/CD)
kairos --json validate

# Validar con config específica
kairos --config ./my-config.json validate
```

**Salida ejemplo:**
```
Kairos Configuration Validation
================================

✓ config
✓ llm: ollama (llama3.2)
✓ mcp:filesystem: http: 14 tools available
✓ governance: 2 policies configured
✓ skill:pdf-processing: Extract text and tables from PDFs...

✓ All checks passed
```

**Códigos de salida:**
- `0`: Todas las verificaciones pasaron
- `1`: Al menos una verificación falló

---

## Comandos de Ejecución

### `kairos run`

Ejecuta un agente de forma interactiva o con un prompt único.

**Flags:**
- `--prompt <text>`: Ejecutar con un único prompt (no interactivo)
- `--profile <name>`: Cargar perfil de config (busca `./config/config.<name>.yaml`)
- `--agent <id>`: ID del agente (default: `kairos-agent`)
- `--role <text>`: Rol del agente (default: `Helpful Assistant`)
- `--skills <dir>`: Directorio de skills a cargar
- `--interactive=false`: Modo pipe (lee de stdin)
- `--no-telemetry`: Deshabilitar salida de telemetría
- `--watch`: Hot-reload de config cuando el archivo cambie

**Modos de ejecución:**

1. **Prompt único** - Ejecuta y termina:
```bash
kairos run --prompt "Explain AI agents"
```

2. **Interactivo (REPL)** - Conversación continua:
```bash
kairos run
# > Hello
# > /tools
# > /help
# > exit
```

3. **Modo pipe** - Lee de stdin:
```bash
echo "What is 2+2?" | kairos run --interactive=false
```

**Comandos REPL:**
- `/help` - Mostrar ayuda
- `/tools` - Listar herramientas disponibles
- `/skills` - Listar skills cargados
- `/exit` - Salir

**Ejemplos:**
```bash
# Prompt único con output limpio
kairos run --no-telemetry --prompt "What is Kairos?"

# JSON output para scripting
kairos --json run --no-telemetry --prompt "List 3 colors"

# Con perfil de desarrollo
kairos run --profile dev

# Con skills personalizados
kairos run --skills ./my-skills

# REPL interactivo
kairos run --no-telemetry

# Con hot-reload de config (desarrollo)
kairos run --watch --profile dev
```

---

## Comandos Operativos

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

---

## Comandos de Introspección

### `kairos explain`

Muestra la configuración y componentes del agente en forma de árbol.

**Flags:**
- `--agent <id>`: Agent ID a inspeccionar (default: `kairos-agent`)
- `--skills <dir>`: Directorio de skills a cargar

**Ejemplo:**
```bash
kairos explain
```

**Salida:**
```
Agent: kairos-agent
├── LLM: ollama (llama3.2)
├── Memory: inmemory
├── Governance: enabled
│   ├── Policy: be-brief
│   └── Policy: no-secrets
├── Tools: 3
│   ├── get_weather (MCP: weather-server)
│   ├── search_docs (MCP: filesystem)
│   └── send_email (MCP: email-server)
├── Skills: 2
│   ├── pdf-processing: Extract text and tables from PDFs
│   └── summarization: Summarize long documents
└── A2A: disabled
```

### `kairos graph`

Genera visualizaciones del grafo del planner en varios formatos.

**Flags:**
- `--path <file>`: Ruta al archivo YAML/JSON del grafo (requerido)
- `--output <format>`: Formato de salida: `mermaid`, `dot`, `json` (default: `mermaid`)

**Ejemplos:**
```bash
# Generar diagrama Mermaid
kairos graph --path workflow.yaml

# Generar Graphviz DOT
kairos graph --path workflow.yaml --output dot

# Exportar como JSON
kairos graph --path workflow.yaml --output json
```

**Salida Mermaid:**
```
graph TD
    detect_intent[detect_intent: detect_intent]
    knowledge[knowledge: knowledge]
    detect_intent --> knowledge
    style detect_intent fill:#90EE90
```

### `kairos adapters list`

Lista los adaptadores disponibles (providers y backends).

**Flags:**
- `--type <type>`: Filtrar por tipo: `llm`, `memory`, `mcp`, `a2a`, `telemetry`

**Ejemplos:**
```bash
# Listar todos los adapters
kairos adapters list

# Solo LLM providers
kairos adapters list --type llm
```

### `kairos adapters info <name>`

Muestra detalles de configuración de un adapter específico.

**Ejemplo:**
```bash
kairos adapters info ollama
```

**Salida:**
```
Adapter: ollama
Type: llm
Description: Local LLM inference with Ollama

Configuration:
  • llm.provider=ollama
  • llm.base_url
  • llm.model

Documentation: https://ollama.ai
```

---

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
