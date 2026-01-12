# Architecture - Kairos Framework

## Goals
- Go-native runtime with first-class SDK.
- Interoperability by standards: MCP, A2A/ACP, AgentSkills, AGENTS.md.
- Observability by default with OpenTelemetry traces, metrics, and logs.
- Multi-agent distributed execution with governance and production readiness.

## Layered Architecture
1) Interfaces (UI/CLI)
2) Control Plane (API, auth, policies, governance)
3) Multi-Agent Runtime Core (Go)
4) Planner + Memory + Tools
5) Interop: MCP + A2A/ACP + AGENTS.md
6) Observability + Storage

## Core Components (Go)
- Agent runtime: lifecycle, scheduling, context propagation, tool execution.
- Planner: explicit graphs + emergent planner, single internal model.
- Memory: short, long, shared, persistent.
- Tools/Skills: AgentSkills as semantic layer; binding to MCP tools.
- Policy engine: scopes, allow/deny, audit events.

## Interoperability
- MCP client and server.
- A2A/ACP for discovery, delegation, and remote execution.
- AGENTS.md auto-loading on startup to enforce repo rules.

### A2A implementation (current)
- gRPC binding with streaming (SendMessage, SendStreamingMessage, GetTask, ListTasks, CancelTask).
- Go types generated directly from `pkg/a2a/proto/a2a.proto` (`scripts/gen-a2a.sh`).
- AgentCard publishing + discovery, plus A2AService server/client.
- Task/Message/Artifact mapping with streaming responses.
- HTTP+JSON and JSON-RPC server bindings (`pkg/a2a/httpjson`, `pkg/a2a/jsonrpc`) with SSE for streaming.
- Task store + push config store backends (in-memory + SQLite).
- Demo multi-agent flow (demoKairos) exercising delegation (orchestrator -> knowledge/spreadsheet).

### A2A storage backends
- In-memory stores: `MemoryTaskStore`, `MemoryPushConfigStore` (default in handlers).
- SQLite stores (no CGO): `SQLiteTaskStore`, `SQLitePushConfigStore` via `modernc.org/sqlite`.
- Schema is created automatically on startup; tasks/configs are stored as JSON blobs with indexes for status, context, and updated time.
- Pagination uses stable ordering: `updated_at DESC`, then `id ASC`.

## Observability
- OpenTelemetry tracing for agent runs, planner steps, tool calls, A2A hops (trace propagation over A2A is implemented).
- Metrics: latency per step, errors per agent, token usage.
- Structured logs with trace/span ids and decision summaries (rationale + inputs/outputs).
- Decision events emitted per iteration, including tool-call outcomes for auditing.

### Telemetry Configuration (OTLP)
Example config block for OTLP exporter:

```json
{
  "telemetry": {
    "exporter": "otlp",
    "otlp_endpoint": "localhost:4317",
    "otlp_insecure": true
  }
}
```

Equivalent environment variables:

- `KAIROS_TELEMETRY_EXPORTER`
- `KAIROS_TELEMETRY_OTLP_ENDPOINT`
- `KAIROS_TELEMETRY_OTLP_INSECURE`
- `KAIROS_TELEMETRY_OTLP_TIMEOUT_SECONDS`

#### Verification Steps
1) Start an OTLP-compatible backend (e.g., local collector on `localhost:4317`).
2) Run an example with OTLP enabled:

```bash
KAIROS_TELEMETRY_EXPORTER=otlp \
KAIROS_TELEMETRY_OTLP_ENDPOINT=localhost:4317 \
KAIROS_TELEMETRY_OTLP_INSECURE=true \
go run ./examples/basic-agent
```

3) Confirm traces and metrics arrive in the backend.

Optional OTLP smoke test:
```bash
KAIROS_OTLP_SMOKE_TEST=1 \
KAIROS_TELEMETRY_OTLP_ENDPOINT=localhost:4317 \
KAIROS_TELEMETRY_OTLP_INSECURE=true \
KAIROS_TELEMETRY_OTLP_TIMEOUT_SECONDS=30 \
go test ./pkg/telemetry -run TestOTLPSmoke -count=1
```

## Data Model (high level)
- Agent: id, role, skills, tools, memory, policies.
- Skill: semantic capability (AgentSkills spec).
- Tool: MCP implementation that fulfills skills.
- Plan: graph or emergent plan state.
- Memory: interface Store/Retrieve with pluggable backends.

## Explicit planner groundwork
- Graph schema (`pkg/planner`): nodes, edges, and optional start node.
- JSON/YAML parsers with validation for well-formed graphs.
- Executor supports per-node tracing with branching/conditions and multi-edge evaluation.

## Execution Flow (runtime)
1) Load AGENTS.md and apply repository rules.
2) Initialize agent with skills, memory, tools, policies.
3) Build plan (explicit graph or emergent).
4) Execute steps with context propagation.
5) Emit traces/metrics/logs and audit events.

## Configuration sources
- File: `~/.kairos/settings.json` or `./.kairos/settings.json`.
- Env: `KAIROS_*` (maps to config keys).
- CLI: `--config=/path/to/settings.json` and repeatable `--set key=value`.
Precedence: defaults < file < env < CLI.
Examples:
```bash
go run ./examples/basic-agent --config=./.kairos/settings.json \
  --set llm.provider=ollama \
  --set telemetry.exporter=stdout
```
See `docs/CONFIGURATION.md` for the full guide.

## Event taxonomy
- Stable semantic events for streaming/logs: `docs/EVENT_TAXONOMY.md`.

## Agent loop options
- `agent.WithDisableActionFallback(true)` disables legacy "Action:" parsing in the ReAct loop when tool calls are supported.
- `agent.WithActionFallbackWarning(true)` emits a warning log when legacy Action parsing is used.
- Config: `agent.disable_action_fallback` or `KAIROS_AGENT_DISABLE_ACTION_FALLBACK=true` (default: true).
- Per-agent overrides can be defined under `agents.<agent_id>`.

### Action fallback deprecation plan
- Phase 1 (current): fallback is disabled by default; enable explicitly via config/env.
- Phase 2 (next minor): warning on every fallback use + doc/changelog note.
- Phase 3 (following minor): legacy-only; requires explicit flag and logs a startup warning.
- Phase 4 (next major): remove fallback path and related flags.

Activation summary:
- Enable fallback only by setting `agent.disable_action_fallback=false` (or `KAIROS_AGENT_DISABLE_ACTION_FALLBACK=false`).

Example config:
```json
{
  "agent": {
    "disable_action_fallback": false,
    "warn_on_action_fallback": true
  },
  "agents": {
    "mcp-agent": {
      "disable_action_fallback": true
    }
  }
}
```

Example config (full, with telemetry):
```json
{
  "agent": {
    "disable_action_fallback": false
  },
  "agents": {
    "mcp-agent": {
      "disable_action_fallback": true
    }
  },
  "telemetry": {
    "exporter": "otlp",
    "otlp_endpoint": "localhost:4317",
    "otlp_insecure": true
  }
}
```

## Governance and Security
- Policy enforcement on tool usage and agent delegation.
- Human-in-the-loop points.
- Auditing for every action and tool call.

## Deployment
- Single Go binary.
- Docker/Kubernetes ready.
- Horizontal scaling with A2A federation.

## Suggested Package Layout (initial)
- core/agent
- core/runtime
- core/planner
- core/memory
- core/tools
- interop/mcp
- interop/a2a
- observability/otel
- controlplane/api
- ui (future)

## Explicit planner walkthrough
- `docs/walkthrough-explicit-planner.md`

## Demo A2A agents walkthrough
- `docs/walkthrough-demo-a2a-agents.md`

## A2A HTTP+JSON and JSON-RPC walkthrough
- `docs/walkthrough-a2a-httpjson-jsonrpc.md`

## Governance walkthrough
- `docs/walkthrough-governance-agentsmd.md`

## Governance usage
- `docs/governance-usage.md`
