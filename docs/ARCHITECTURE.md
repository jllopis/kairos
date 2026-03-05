# Arquitectura - Kairos

<!--
Copyright 2026 © The Kairos Authors
SPDX-License-Identifier: Apache-2.0
-->

## Visión general

Kairos es un **framework de agentes IA nativo en Go** diseñado para entornos
de producción. Combina bucles ReAct emergentes, planificadores declarativos,
protocolos de interoperabilidad (MCP, A2A), observabilidad nativa (OpenTelemetry)
y gobernanza integrada bajo una única API coherente.

## Arquitectura por capas

El framework se organiza en 6 capas, desde la interfaz de usuario hasta el
almacenamiento:

```mermaid
graph TB
    subgraph "Capa 1 — Interfaces"
        CLI["CLI (kairos run / init / validate)"]
        API["Control Plane API (kairosctl)"]
    end

    subgraph "Capa 2 — Control Plane"
        Auth["Auth & Policies"]
        Gov["Governance Engine"]
        Guard["Guardrails"]
    end

    subgraph "Capa 3 — Runtime"
        Agent["Agent Runtime"]
        Planner["Planner (DAG / Emergente)"]
        Skills["Skills & Tools"]
    end

    subgraph "Capa 4 — Interop"
        MCP["MCP Client/Server"]
        A2A["A2A gRPC / HTTP / JSON-RPC"]
        Conn["Connectors (OpenAPI, GraphQL, gRPC, SQL)"]
    end

    subgraph "Capa 5 — LLM & Memoria"
        LLM["LLM Providers (OpenAI, Anthropic, Gemini, Qwen, Ollama)"]
        Mem["Memory (Vector, Conversation, Semantic)"]
    end

    subgraph "Capa 6 — Observabilidad & Storage"
        OTEL["OpenTelemetry (Trazas, Métricas, Logs)"]
        Store["Storage (SQLite, In-Memory, Qdrant)"]
    end

    CLI --> Agent
    API --> Agent
    Agent --> Auth
    Agent --> Gov
    Agent --> Guard
    Agent --> Planner
    Agent --> Skills
    Agent --> LLM
    Agent --> Mem
    Skills --> MCP
    Skills --> Conn
    Agent --> A2A
    Agent --> OTEL
    Mem --> Store
    A2A --> Store
```

## Diagrama C4 — Contexto del sistema

```mermaid
C4Context
    title Kairos — Diagrama de contexto

    Person(dev, "Desarrollador", "Construye agentes IA con Go")
    Person(ops, "Operador", "Despliega y monitoriza agentes")

    System(kairos, "Kairos Framework", "Framework Go para agentes IA con ReAct, planner DAG, MCP, A2A y OTEL")
    System(kairosctl, "kairosctl", "Plano de control: observabilidad, evaluación, despliegue")

    System_Ext(llm, "LLM Providers", "OpenAI, Anthropic, Gemini, Qwen, Ollama")
    System_Ext(mcp_server, "MCP Servers", "Servidores de herramientas MCP externos")
    System_Ext(a2a_agents, "Agentes remotos", "Otros agentes via A2A")
    System_Ext(otel_backend, "OTEL Backend", "Jaeger, Prometheus, Grafana")
    System_Ext(vector_db, "Vector DB", "Qdrant u otros backends de embeddings")

    Rel(dev, kairos, "Construye agentes", "Go SDK")
    Rel(ops, kairosctl, "Gestiona agentes", "CLI / HTTP")
    Rel(kairosctl, kairos, "Importa", "Go module")
    Rel(kairos, llm, "Envía prompts", "HTTPS")
    Rel(kairos, mcp_server, "Descubre y ejecuta tools", "MCP (stdio / HTTP)")
    Rel(kairos, a2a_agents, "Delega tareas", "gRPC / HTTP / JSON-RPC")
    Rel(kairos, otel_backend, "Exporta trazas y métricas", "OTLP gRPC")
    Rel(kairos, vector_db, "Almacena embeddings", "HTTP / gRPC")
```

## Diagrama C4 — Contenedores

```mermaid
C4Container
    title Kairos — Diagrama de contenedores

    Person(dev, "Desarrollador")

    Container_Boundary(kairos, "Kairos Framework") {
        Container(agent, "Agent Runtime", "Go", "Bucle ReAct + planner DAG, resolución de tools, guardrails")
        Container(planner, "Planner", "Go", "Ejecución de grafos DAG con branching condicional")
        Container(llm_pkg, "LLM Package", "Go", "Interfaces Provider/StreamingProvider y tipos compartidos")
        Container(mcp_pkg, "MCP Package", "Go", "Cliente MCP con pool, retry, caché de tools")
        Container(a2a_pkg, "A2A Package", "Go", "Cliente/servidor A2A (gRPC, HTTP/JSON, JSON-RPC)")
        Container(memory, "Memory", "Go", "Vector store, conversation memory, estrategias de truncación")
        Container(governance, "Governance", "Go", "PolicyEngine, RuleSet, ApprovalHook, ToolFilter")
        Container(guardrails, "Guardrails", "Go", "InputChecker, OutputFilter (PII, prompt injection)")
        Container(resilience, "Resilience", "Go", "Retry, CircuitBreaker, Timeout, Fallback")
        Container(telemetry, "Telemetry", "Go", "OTEL init, métricas, atributos semánticos, logging")
        Container(connectors, "Connectors", "Go", "OpenAPI, GraphQL, gRPC, SQL → core.Tool[]")
    }

    System_Ext(llm, "LLM APIs")
    System_Ext(mcp_ext, "MCP Servers")
    System_Ext(a2a_ext, "Agentes remotos")
    System_Ext(otel, "OTEL Backend")

    Rel(dev, agent, "Crea y ejecuta agentes", "Go API")
    Rel(agent, planner, "Ejecuta plan DAG")
    Rel(agent, llm_pkg, "Envía ChatRequest")
    Rel(agent, mcp_pkg, "Resuelve tools MCP")
    Rel(agent, a2a_pkg, "Delega a agentes remotos")
    Rel(agent, memory, "Store/Retrieve contexto")
    Rel(agent, governance, "Evalúa políticas")
    Rel(agent, guardrails, "Filtra input/output")
    Rel(agent, resilience, "Retry/CB en operaciones")
    Rel(agent, telemetry, "Emite trazas/métricas")
    Rel(agent, connectors, "Genera tools desde specs")
    Rel(llm_pkg, llm, "HTTPS")
    Rel(mcp_pkg, mcp_ext, "stdio / HTTP")
    Rel(a2a_pkg, a2a_ext, "gRPC / HTTP")
    Rel(telemetry, otel, "OTLP gRPC")
```

## Interfaces core

Las interfaces principales definen el contrato entre componentes:

```go
// Agent define la interfaz de un agente ejecutable.
type Agent interface {
    ID() string
    Role() string
    Skills() []Skill
    Memory() Memory
    Run(ctx context.Context, input any) (any, error)
}

// Tool define una herramienta invocable por el agente.
type Tool interface {
    Name() string
    Call(ctx context.Context, input any) (any, error)
    ToolDefinition() llm.Tool
}

// Memory define almacenamiento de contexto.
type Memory interface {
    Store(ctx context.Context, data any) error
    Retrieve(ctx context.Context, query any) (any, error)
}

// Provider define la interfaz para backends LLM.
type Provider interface {
    Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
}

// StreamingProvider extiende Provider con streaming.
type StreamingProvider interface {
    Provider
    ChatStream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error)
}
```

## Flujo del Agent Loop (ReAct)

El bucle ReAct es el mecanismo principal de ejecución emergente:

```mermaid
flowchart TD
    Start([Input del usuario]) --> GuardIn{Guardrails\nInput Check}
    GuardIn -->|Bloqueado| Reject([Error: input rechazado])
    GuardIn -->|OK| Resolve[Resolver tools:\nlocal + skills + MCP + connectors]
    Resolve --> Filter[Filtrar por governance]
    Filter --> SysPrompt[Construir system prompt:\nrole + AGENTS.md + tool defs]
    SysPrompt --> LoadMem[Cargar memoria:\nconversation + semantic]
    LoadMem --> LLMCall[Llamar LLM.Chat\ncon mensajes + tools]

    LLMCall --> HasTools{¿Respuesta tiene\nToolCalls?}

    HasTools -->|Sí| PolicyCheck{Policy Engine\n¿Permite ejecución?}
    PolicyCheck -->|Deny| PolicyErr([Error: política denegada])
    PolicyCheck -->|Approve| ExecTool[Ejecutar tool.Call]
    ExecTool --> AppendResult[Append resultado\ncomo mensaje tool]
    AppendResult --> IterCheck{¿Max iteraciones\nalcanzadas?}
    IterCheck -->|No| LLMCall
    IterCheck -->|Sí| Timeout([Error: max iterations])

    HasTools -->|No| HasAnswer{¿Tiene contenido\nfinal?}
    HasAnswer -->|Sí| GuardOut{Guardrails\nOutput Filter}
    GuardOut -->|Redactado| Return([Respuesta filtrada])
    GuardOut -->|OK| StoreMem[Almacenar en memoria]
    StoreMem --> Return2([Respuesta final])
    HasAnswer -->|No| LLMCall
```

## Flujo del Planner (DAG explícito)

Cuando se configura un grafo explícito con `agent.WithPlanner(graph)`, el
agente ejecuta un DAG en lugar del bucle ReAct:

```mermaid
flowchart TD
    Start([Input]) --> ParseGraph[Parsear grafo\nJSON / YAML]
    ParseGraph --> Validate[Validar:\nnodos, edges, ciclos]
    Validate --> InitState[Inicializar State:\ninput, outputs map]
    InitState --> FindStart[Resolver nodo inicio]

    FindStart --> ExecNode[Ejecutar nodo actual]

    ExecNode --> NodeType{Tipo de nodo}

    NodeType -->|tool| ExecTool[Handler: ejecutar tool\npor nombre]
    NodeType -->|llm| ExecLLM[Handler: llamar LLM\ncon input]
    NodeType -->|agent| ExecAgent[Handler: ejecutar\nsub-agente]
    NodeType -->|decision| ExecDecision[Handler: evaluar\ncondición]
    NodeType -->|noop| ExecNoop[Handler: pass-through]

    ExecTool --> StoreOutput
    ExecLLM --> StoreOutput
    ExecAgent --> StoreOutput
    ExecDecision --> StoreOutput
    ExecNoop --> StoreOutput

    StoreOutput[Guardar output en State\n+ emitir AuditEvent] --> FindEdges[Buscar edges salientes]

    FindEdges --> EvalCond{Evaluar condiciones\nen edges}
    EvalCond -->|Match| NextNode[Siguiente nodo]
    EvalCond -->|Sin match| End([Fin: retornar\núltimo output])

    NextNode --> CycleCheck{¿Ciclo detectado?}
    CycleCheck -->|No| ExecNode
    CycleCheck -->|Sí| CycleErr([Error: ciclo en grafo])
```

### Condiciones soportadas en edges

| Patrón | Descripción |
|--------|-------------|
| `last==valor` | Último output igual a valor |
| `last!=valor` | Último output distinto de valor |
| `last.contains:texto` | Último output contiene texto |
| `output.nodeID==valor` | Output de nodo específico igual a valor |
| (vacío) | Edge por defecto (siempre se toma) |

## Interoperabilidad: MCP

El protocolo MCP permite descubrir y ejecutar herramientas externas:

```mermaid
sequenceDiagram
    participant Agent
    participant MCPClient as MCP Client
    participant MCPPool as MCP Pool
    participant MCPServer as MCP Server externo

    Agent->>MCPClient: ListTools()
    MCPClient->>MCPPool: GetConnection(server)
    MCPPool->>MCPServer: tools/list
    MCPServer-->>MCPPool: Tool definitions
    MCPPool-->>MCPClient: Connection + tools
    MCPClient->>MCPClient: Cache tools (TTL)
    MCPClient-->>Agent: []core.Tool (via ToolAdapter)

    Note over Agent: LLM elige tool call

    Agent->>MCPClient: CallTool(name, args)
    MCPClient->>MCPPool: GetConnection(server)
    MCPPool->>MCPServer: tools/call
    MCPServer-->>MCPPool: Resultado
    MCPPool-->>MCPClient: Resultado
    MCPClient-->>Agent: Tool result
```

### Componentes MCP

| Componente | Paquete | Descripción |
|------------|---------|-------------|
| `Client` | `pkg/mcp` | Cliente MCP (Stdio + StreamableHTTP), retry exponencial, caché de tools |
| `Server` | `pkg/mcp` | Servidor MCP con `RegisterTool()`, sirve via Stdio o HTTP |
| `ToolAdapter` | `pkg/mcp` | Implementa `core.Tool` envolviendo herramientas MCP |
| `Pool` | `pkg/mcp/pool` | Pool de conexiones con reference counting, health check, idle cleanup |

## Interoperabilidad: A2A (Agent-to-Agent)

El protocolo A2A permite arquitecturas distribuidas entre agentes:

```mermaid
sequenceDiagram
    participant AgentA as Agente Local
    participant A2AClient as A2A Client
    participant A2AServer as A2A Server (remoto)
    participant Handler as SimpleHandler
    participant Executor as Executor
    participant TaskStore as TaskStore

    AgentA->>A2AClient: SendMessage(message)
    A2AClient->>A2AServer: gRPC SendMessage
    A2AServer->>Handler: SendMessage(ctx, msg)
    Handler->>TaskStore: CreateTask()
    Handler->>Executor: Run(ctx, message)
    Executor-->>Handler: (output, artifacts)
    Handler->>TaskStore: UpdateStatus(completed)
    Handler-->>A2AServer: Task result
    A2AServer-->>A2AClient: Task response
    A2AClient-->>AgentA: Task result

    Note over A2AClient,A2AServer: También soporta streaming,<br/>push notifications y approval flow
```

### Transportes A2A

| Transporte | Paquete | Protocolo |
|------------|---------|-----------|
| gRPC | `pkg/a2a/server`, `pkg/a2a/client` | Protobuf + streaming bidireccional |
| HTTP/JSON | `pkg/a2a/httpjson` | REST + SSE para streaming |
| JSON-RPC | `pkg/a2a/jsonrpc` | JSON-RPC 2.0 + SSE |

### Backends de almacenamiento A2A

| Store | Implementación | Descripción |
|-------|---------------|-------------|
| `MemoryTaskStore` | In-memory | Por defecto, para desarrollo |
| `SQLiteTaskStore` | SQLite (sin CGO) | Producción, via `modernc.org/sqlite` |
| `MemoryPushConfigStore` | In-memory | Push notifications (dev) |
| `SQLitePushConfigStore` | SQLite | Push notifications (prod) |

## LLM Providers

El sistema de providers sigue una arquitectura pluggable:

```mermaid
graph TB
    Agent["Agent Runtime"] --> ProviderIF["llm.Provider interface<br/>Chat(ctx, ChatRequest) → ChatResponse"]

    ProviderIF --> Ollama["Ollama Provider<br/>pkg/llm/ollama.go"]
    ProviderIF --> OpenAI["OpenAI Provider<br/>providers/openai/"]
    ProviderIF --> Anthropic["Anthropic Provider<br/>providers/anthropic/"]
    ProviderIF --> Gemini["Gemini Provider<br/>providers/gemini/"]
    ProviderIF --> Qwen["Qwen Provider<br/>providers/qwen/"]

    style ProviderIF fill:#f9f,stroke:#333
```

Cada provider traduce entre los tipos genéricos (`ChatRequest`/`ChatResponse`)
y el formato nativo del LLM. Todos soportan `StreamingProvider` para
respuestas en tiempo real.

### Tipos compartidos (`pkg/llm/provider.go`)

| Tipo | Descripción |
|------|-------------|
| `ChatRequest` | Modelo, mensajes, tools, temperatura |
| `ChatResponse` | Contenido, tool_calls, usage |
| `Tool` | Definición de función (type + function) |
| `ToolCall` | Llamada a tool del LLM (id, type, function) |
| `Message` | Role, content, tool_calls, tool_call_id |
| `StreamChunk` | Delta de contenido para streaming |

## Conectores (Tools)

Los conectores generan `[]core.Tool` desde especificaciones externas:

```mermaid
graph LR
    subgraph "Specs externas"
        OAS["OpenAPI 3.x / Swagger 2.0"]
        GQL["GraphQL Schema"]
        Proto[".proto files"]
        DB["Database Schema"]
    end

    subgraph "Conectores"
        OAPI["OpenAPIConnector"]
        GQLC["GraphQLConnector"]
        GRPC["GRPCConnector"]
        SQL["SQLConnector"]
    end

    subgraph "Output"
        Tools["[]core.Tool"]
    end

    OAS --> OAPI
    GQL --> GQLC
    Proto --> GRPC
    DB --> SQL

    OAPI --> Tools
    GQLC --> Tools
    GRPC --> Tools
    SQL --> Tools

    Tools --> Agent["Agent Runtime"]
```

### Conectores disponibles

| Conector | Spec de entrada | Estado |
|----------|-----------------|--------|
| `OpenAPIConnector` | OpenAPI 3.x, Swagger 2.0 | Implementado |
| `MCPConnector` | MCP protocol | Implementado |
| `GraphQLConnector` | Schema introspection | Implementado |
| `GRPCConnector` | `.proto` files | Implementado |
| `SQLConnector` | Database schema | Implementado |

## Memoria

El sistema de memoria tiene dos dimensiones: **semántica** (vector) y
**conversacional** (historial):

```mermaid
graph TB
    Agent["Agent"] --> VM["VectorMemory<br/>(core.Memory)"]
    Agent --> CM["ConversationMemory"]

    VM --> Embedder["Embedder interface<br/>Embed(text) → []float32"]
    VM --> VectorStore["VectorStore interface<br/>Upsert / Search"]

    Embedder --> OllamaEmb["Ollama Embedder"]
    VectorStore --> Qdrant["Qdrant Store"]
    VectorStore --> InMem["In-Memory Store"]

    CM --> Trunc["TruncationStrategy"]
    Trunc --> Window["WindowStrategy<br/>(últimos N mensajes)"]
    Trunc --> Token["TokenStrategy<br/>(límite de tokens)"]
    Trunc --> Summ["SummarizationStrategy<br/>(resumen via LLM)"]
```

## Gobernanza y guardrails

### Governance

El `PolicyEngine` evalúa cada acción antes de ejecutarla:

```mermaid
flowchart LR
    Action["Action<br/>(tool call, delegation)"] --> PE["PolicyEngine.Evaluate()"]
    PE --> Rules["RuleSet<br/>allow/deny por scope"]
    Rules --> Decision{Decision}
    Decision -->|Allow| Exec["Ejecutar acción"]
    Decision -->|Deny| Block["Bloquear + audit log"]
    Decision -->|RequireApproval| HITL["Human-in-the-Loop<br/>ApprovalHook"]
    HITL -->|Approved| Exec
    HITL -->|Rejected| Block
```

### Guardrails

Filtros de seguridad en input y output:

| Tipo | Interface | Función |
|------|-----------|---------|
| Input | `InputChecker` | Detecta prompt injection, contenido peligroso |
| Output | `OutputFilter` | Redacta PII, contenido sensible |

## Resiliencia

Patrones de resiliencia integrados en el framework:

| Patrón | Tipo | Descripción |
|--------|------|-------------|
| Retry | `RetryConfig` | Exponential backoff con jitter, `IsRecoverable` check |
| Circuit Breaker | `CircuitBreaker` | Closed → Open → HalfOpen, umbrales configurables |
| Timeout | `TimeoutConfig` | Goroutine-based timeout con `CodeTimeout` error |
| Fallback | `FallbackStrategy` | Static, Cached, Chained, `GracefulDegradation` |

## Observabilidad

### Instrumentación OTEL

```mermaid
graph TB
    Agent["Agent Runtime"] --> Tracer["TracerProvider"]
    Agent --> Meter["MeterProvider"]
    Agent --> Logger["slog + TraceHandler"]

    Tracer --> StdoutT["stdout exporter"]
    Tracer --> OTLPT["OTLP gRPC exporter"]

    Meter --> StdoutM["stdout exporter"]
    Meter --> OTLPM["OTLP gRPC exporter"]

    Logger --> Console["Console (trace_id, span_id)"]

    OTLPT --> Backend["OTEL Backend<br/>(Jaeger, Tempo)"]
    OTLPM --> Backend2["OTEL Backend<br/>(Prometheus)"]
```

### Métricas del agente

| Métrica | Tipo | Descripción |
|---------|------|-------------|
| `kairos.agent.run.count` | Counter | Ejecuciones totales del agente |
| `kairos.agent.error.count` | Counter | Errores totales |
| `kairos.agent.run.latency_ms` | Histogram | Latencia por ejecución |
| `kairos.agent.llm.latency_ms` | Histogram | Latencia de llamadas LLM |
| `kairos.agent.tool.latency_ms` | Histogram | Latencia de tool calls |
| `kairos.agent.memory.latency_ms` | Histogram | Latencia de operaciones de memoria |

### Métricas de errores y salud

| Métrica | Tipo | Descripción |
|---------|------|-------------|
| `kairos.errors.total` | Counter | Errores por código y severidad |
| `kairos.errors.recovered` | Counter | Errores recuperados |
| `kairos.errors.rate` | Gauge | Tasa de errores |
| `kairos.health.status` | Gauge | Estado de salud (0=down, 1=degraded, 2=healthy) |
| `kairos.circuitbreaker.state` | Gauge | Estado del circuit breaker |

### Configuración de telemetría

```json
{
  "telemetry": {
    "exporter": "otlp",
    "otlp_endpoint": "localhost:4317",
    "otlp_insecure": true
  }
}
```

Variables de entorno: `KAIROS_TELEMETRY_EXPORTER`, `KAIROS_TELEMETRY_OTLP_ENDPOINT`,
`KAIROS_TELEMETRY_OTLP_INSECURE`, `KAIROS_TELEMETRY_OTLP_TIMEOUT_SECONDS`.

## Configuración

Fuentes de configuración con precedencia creciente:

1. **Valores por defecto** (hardcoded)
2. **Archivo** (`~/.kairos/settings.json` o `./.kairos/settings.json`)
3. **Variables de entorno** (`KAIROS_*`)
4. **CLI flags** (`--config=...`, `--set key=value`)

Ver [CONFIGURATION.md](CONFIGURATION.md) para la guía completa.

## Layout de paquetes

```
pkg/
├── agent/          # Agent runtime (ReAct + planner integration)
├── a2a/            # A2A protocol (client, server, types, agentcard)
│   ├── client/
│   ├── server/
│   ├── httpjson/
│   ├── jsonrpc/
│   ├── agentcard/
│   └── types/
├── config/         # Configuration types
├── connectors/     # External API connectors (OpenAPI, GraphQL, gRPC, SQL)
├── core/           # Shared interfaces (Agent, Tool, Memory, Event, Health)
├── discovery/      # Agent/service discovery
├── errors/         # Typed errors (KairosError, ErrorCode)
├── governance/     # Policy engine, rules, approval hooks
├── guardrails/     # Input/output security filters
├── llm/            # LLM provider interface + Ollama implementation
├── mcp/            # MCP client/server + tool adapter
│   └── pool/       # Connection pooling
├── memory/         # Vector + conversation memory
│   ├── ollama/     # Ollama embedder
│   └── qdrant/     # Qdrant vector store
├── planner/        # Graph-based DAG planner
├── resilience/     # Retry, circuit breaker, timeout, fallback
├── runtime/        # Runtime orchestration
├── skills/         # AgentSkills spec loader
├── telemetry/      # OpenTelemetry setup + metrics + logging
└── testing/        # Test helpers, scenarios, mock provider

providers/
├── openai/         # OpenAI provider (GPT-4, GPT-5)
├── anthropic/      # Anthropic provider (Claude)
├── gemini/         # Google Gemini provider
└── qwen/           # Alibaba Qwen provider (DashScope)

examples/
├── 01-hello-agent/ ... 20-conversation-memory/
├── playbook/       # Narrativa completa multi-agente
└── mcp-http-server/
```

## Enlaces relacionados

- [API Reference](API.md)
- [Planner conceptos](Conceptos_Planner.md)
- [MCP Protocol](protocols/MCP.md)
- [A2A Protocol](protocols/A2A/Overview.md)
- [Error Handling](ERROR_HANDLING.md)
- [Observability](OBSERVABILITY.md)
- [Governance](governance-usage.md)
- [Guardrails](GUARDRAILS.md)
- [Testing](TESTING.md)
- [Playbook](../examples/playbook/README.md)
