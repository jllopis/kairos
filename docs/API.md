# Referencia de API

<!--
Copyright 2026 © The Kairos Authors
SPDX-License-Identifier: Apache-2.0
-->

Esta referencia cubre las APIs públicas del framework. Para cada paquete se
documentan las interfaces, tipos principales y ejemplos de uso.

## Agent

Creación básica:

```go
a, err := agent.New("demo-agent", llmProvider,
  agent.WithRole("Analista"),
  agent.WithTools([]core.Tool{tool}),
  agent.WithMCPClients(client),
)
```

Opciones comunes:

- `agent.WithRole(...)`: rol corto del agente.
- `agent.WithSkills(...)`: habilidades semánticas (skills).
- `agent.WithSkillsFromDir(...)`: carga skills desde un directorio con subcarpetas `SKILL.md`.
- `agent.WithTools(...)`: tools concretas.
- `agent.WithMCPClients(...)`: tools remotas vía MCP.
- `agent.WithMemory(...)`: memoria semántica para recuperar contexto.
- `agent.WithConversationMemory(...)`: memoria de conversación para chat multi-turno.
- `agent.WithToolFilter(...)`: filtrado de tools via governance.
- `agent.WithPolicyEngine(...)`: enforcement de políticas.
- `agent.WithEventEmitter(...)`: eventos semánticos.
- `agent.WithGuardrails(...)`: integra guardrails de entrada/salida en el runtime.
- `agent.WithPlanner(...)`: ejecuta un plan explícito (grafo) en el runtime.
- `agent.WithPlannerHandlers(...)`: handlers custom por tipo de nodo.
- `agent.WithPlannerIDHandlers(...)`: handlers opt-in por `node.id` (sobrescriben el tipo).
- `agent.WithPlannerAuditStore(...)`: persistencia de auditoría del planner.
- `agent.WithPlannerAuditHook(...)`: hook de auditoría en tiempo real.

Los tools locales deben implementar `core.Tool`, incluyendo `ToolDefinition()`,
que devuelve el schema (`llm.Tool`) usado para tool-calling.

Ejecución:

```go
resp, err := a.Run(ctx, "Resuelve esto...")
```

Con sessionID para conversaciones:

```go
ctx := core.WithSessionID(context.Background(), "user-123")
resp, err := a.Run(ctx, "Hola, me llamo Juan")
```

## Planner explícito (runtime)

Ejemplo de ejecución del planner desde el agente:

```go
graph, _ := planner.ParseYAML(planBytes)
agent.New("demo-agent", llmProvider,
  agent.WithPlanner(graph),
)
```

Tipos de nodo soportados por defecto:

- `tool`: ejecuta `node.tool` o `metadata.tool`.
- `agent`: ejecuta el loop emergente del agente.
- `llm`: llamada directa al LLM (sin tools).
- `decision` / `noop`: no-op (mantiene `state.Last`).
- Si `node.type` coincide con el nombre de una tool, se ejecuta esa tool.
  
Aliases soportados (ejemplos legacy): `init`, `validation`, `llm_call`,
`format_output`, `error_handler`, `terminal`.

Resolución de handlers:
- Por defecto se resuelve por `node.type`.
- Si se configuran handlers por `node.id`, estos tienen prioridad sobre el tipo.

Entrada por nodo:
- Si `node.input` no está definido, se usa `state.Last`.
- El input inicial está disponible como `state.Outputs["input"]`.

Condiciones soportadas:
- `last==<valor>` / `last!=<valor>`
- `last.contains:<texto>`
- `output.<node>.<path>==<valor>` / `output.<node>.<path>!=<valor>`
- `output.<node>.<path>.contains:<texto>`

## Task (core)

Crear y propagar una Task:

```go
task := core.NewTask("Resumir ventas", "orchestrator")
ctx = core.WithTask(ctx, task)
```

Cuando el agente ejecuta con una Task en contexto, actualiza estado y resultado
automáticamente. Ver [TASKS.md](TASKS.md) para el detalle.

## LLM Provider

### Interface base

```go
type Provider interface {
    Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
}
```

### Streaming

```go
type StreamingProvider interface {
    Provider
    ChatStream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error)
}

type StreamChunk struct {
    Content    string     // Delta de texto
    ToolCalls  []ToolCall // Tool calls parciales
    Done       bool       // Último chunk
    Error      error      // Error si lo hay
}
```

Los providers de OpenAI, Anthropic, Gemini y Qwen implementan ambas interfaces.

### Providers disponibles

```go
import "github.com/jllopis/kairos/providers/openai"

provider := openai.New(
    openai.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
    openai.WithModel("gpt-4"),
)
```

| Provider | Import | Constructor | Modelo por defecto |
|----------|--------|-------------|-------------------|
| OpenAI | `providers/openai` | `openai.New(opts...)` | `gpt-5-mini` |
| Anthropic | `providers/anthropic` | `anthropic.New(opts...)` | `claude-haiku-4-20250514` |
| Gemini | `providers/gemini` | `gemini.New(opts...)` | `gemini-3-flash-preview` |
| Qwen | `providers/qwen` | `qwen.New(opts...)` | `qwen-turbo` |
| Ollama | `pkg/llm` | `llm.NewOllamaProvider(url, model)` | (configurable) |

### Mock providers (testing)

```go
// Respuesta fija
llmProvider := &llm.MockProvider{Response: "ok"}

// Secuencia de respuestas scripted
llmProvider := llm.NewScriptedMockProvider([]llm.ChatResponse{
    {Content: "Paso 1"},
    {Content: "Final Answer: resultado"},
})
```

## Memory

### Semantic Memory (Vector)

```go
store, _ := qdrant.New("localhost:6334")
embedder := ollama.NewEmbedder("http://localhost:11434", "nomic-embed-text")
mem, _ := memory.NewVectorMemory(context.Background(), store, embedder, "kairos")
_ = mem.Initialize(context.Background())

_ = mem.Store(ctx, "Mi color favorito es azul.")
matches, _ := mem.Retrieve(ctx, "color favorito")
```

### Conversation Memory (Chat History)

```go
convMem := memory.NewInMemoryConversation(memory.ConversationConfig{
    TruncationStrategy: memory.NewWindowStrategy(20, true),
})

a, _ := agent.New("chat-agent", llmProvider,
    agent.WithConversationMemory(convMem),
)

ctx := core.WithSessionID(context.Background(), "session-123")
a.Run(ctx, "Hola, me llamo Juan")
a.Run(ctx, "¿Cómo me llamo?")  // Recuerda "Juan"
```

Backends disponibles:
- `memory.NewInMemoryConversation(...)`: desarrollo/testing
- `memory.NewFileConversation(...)`: persistencia en archivos
- `memory.NewPostgresConversation(...)`: producción distribuida

Estrategias de truncado:
- `memory.NewWindowStrategy(n, keepSystem)`: últimos N mensajes
- `memory.NewTokenStrategy(n, keepSystem)`: máximo N tokens
- `memory.NewSummarizationStrategy(...)`: resume mensajes antiguos

Ver [CONVERSATION_MEMORY.md](CONVERSATION_MEMORY.md) para documentación completa.

## MCP

Cliente MCP por stdio:

```go
client, _ := mcp.NewClientWithStdioProtocol("node", []string{serverPath}, "2024-11-05")
agent.New("demo-agent", llmProvider,
  agent.WithMCPClients(client),
)
```

Cliente MCP por HTTP:

```go
client, _ := mcp.NewClientWithStreamableHTTPProtocol(serverURL, "2024-11-05")
```

Opciones del cliente:

- `mcp.WithPolicyEngine(pe)`: aplica governance a tool calls.
- `mcp.WithToolCacheTTL(dur)`: TTL para caché de tool definitions.
- `mcp.WithRetryConfig(cfg)`: retry con backoff exponencial.

### MCP Pool

Pool de conexiones para escenarios multi-agente:

```go
p := pool.New(
    pool.WithMaxConnsPerServer(5),
    pool.WithHealthCheckInterval(30 * time.Second),
    pool.WithIdleTimeout(5 * time.Minute),
)

conn, _ := p.Get(ctx, serverConfig)
defer p.Release(conn)
```

### MCP Server

```go
srv := mcp.NewServer("my-server", "1.0.0")
srv.RegisterTool("greet", "Saluda al usuario", schema, handler)
srv.ServeStdio()              // o
srv.ServeStreamableHTTP(addr) // para HTTP
```

## A2A (Agent-to-Agent)

### Server

```go
handler := &server.SimpleHandler{
  Store:         server.NewMemoryTaskStore(),
  Executor:      myExecutor{},
  Card:          myAgentCard(),
  PushCfgs:      server.NewMemoryPushConfigStore(),
  ApprovalStore: server.NewMemoryApprovalStore(),
}
```

### Client

```go
client, _ := a2a.NewClient("localhost:50051",
    a2a.WithTimeout(30 * time.Second),
    a2a.WithRetry(3),
)

task, _ := client.SendMessage(ctx, message)
```

Para bindings y transportes, ver [A2A Overview](protocols/A2A/Overview.md).

## Connectors

Los conectores generan `[]core.Tool` a partir de especificaciones externas.

### Interface

```go
type Connector interface {
    Tools() []core.Tool
    Execute(ctx context.Context, toolName string, args map[string]any) (any, error)
}
```

### OpenAPI Connector

Convierte especificaciones OpenAPI/Swagger en tools ejecutables:

```go
import "github.com/jllopis/kairos/pkg/connectors"

// Desde archivo
conn, _ := connectors.NewOpenAPIConnectorFromFile("petstore.yaml",
    connectors.WithBaseURL("https://api.example.com"),
    connectors.WithBearerToken(os.Getenv("API_TOKEN")),
)

// Desde URL
conn, _ := connectors.NewOpenAPIConnectorFromURL("https://api.example.com/openapi.yaml")

// Desde bytes
conn, _ := connectors.NewOpenAPIConnectorFromBytes(specData)
```

Opciones de autenticación:
- `WithAPIKey(key, header)`: API key en header personalizado
- `WithBearerToken(token)`: Bearer token en Authorization
- `WithBasicAuth(user, pass)`: HTTP Basic Auth
- `WithHTTPClient(client)`: cliente HTTP personalizado
- `WithBaseURL(url)`: URL base para las llamadas

Uso con el agente:

```go
tools := conn.Tools() // []core.Tool generados desde la spec

a, _ := agent.New("api-agent", llmProvider,
    agent.WithTools(tools),
)
```

### GraphQL Connector

```go
conn, _ := connectors.NewGraphQLConnector("https://api.example.com/graphql",
    connectors.WithGraphQLHeaders(map[string]string{"Authorization": "Bearer ..."}),
)

// O desde un schema ya cargado
conn, _ := connectors.NewGraphQLConnectorFromSchema(endpoint, schema,
    connectors.WithGraphQLToolPrefix("gql"),
)
```

### gRPC Connector

```go
conn, _ := connectors.NewGRPCConnector("localhost:50051",
    connectors.WithGRPCInsecure(true),
    connectors.WithGRPCToolPrefix("rpc"),
)

// O desde descriptores de servicios
conn, _ := connectors.NewGRPCConnectorFromServices(target, services,
    connectors.WithGRPCDialOptions(grpc.WithTransportCredentials(...)),
)
```

### SQL Connector

```go
conn, _ := connectors.NewSQLConnector(db, "postgres",
    connectors.WithSQLTables("users", "orders"),
    connectors.WithSQLReadOnly(true),
    connectors.WithSQLToolPrefix("db"),
)

// O desde definiciones de tabla
conn, _ := connectors.NewSQLConnectorFromTables(db, "postgres", tables)
```

Ver [CONNECTORS.md](CONNECTORS.md) para documentación completa y el ejemplo
[17-openapi-connector](../examples/17-openapi-connector/README.md).

## Guardrails

Filtros de seguridad para input y output del agente.

### Interfaces

```go
type InputChecker interface {
    CheckInput(ctx context.Context, input string) CheckResult
    ID() string
}

type OutputFilter interface {
    FilterOutput(ctx context.Context, output string) FilterResult
    ID() string
}
```

### Resultado de checks

```go
type CheckResult struct {
    Blocked bool
    Reason  string
    Details map[string]any
}

type FilterResult struct {
    Output     string      // Texto filtrado
    Redactions []Redaction // Lista de redacciones aplicadas
}
```

### Guardrails orchestrator

```go
import "github.com/jllopis/kairos/pkg/guardrails"

g := guardrails.New(
    guardrails.WithInputChecker(guardrails.NewPromptInjectionDetector()),
    guardrails.WithInputChecker(guardrails.NewContentFilter(
        guardrails.CategoryViolence,
        guardrails.CategoryHate,
    )),
    guardrails.WithOutputFilter(guardrails.NewPIIFilter(
        guardrails.PIIModeRedact,
        guardrails.PIITypeEmail,
        guardrails.PIITypePhone,
        guardrails.PIITypeSSN,
    )),
)

// Integrar con el agente
a, _ := agent.New("safe-agent", llmProvider,
    agent.WithGuardrails(g),
)
```

### Checkers incluidos

| Checker | Tipo | Descripción |
|---------|------|-------------|
| `PromptInjectionDetector` | Input | Detecta intentos de prompt injection |
| `ContentFilter` | Input | Filtra 12 categorías de contenido (violencia, odio, sexual, etc.) |
| `PIIFilter` | Output | Redacta 9 tipos de PII (email, teléfono, SSN, tarjeta, etc.) |

### Modos de PIIFilter

| Modo | Descripción |
|------|-------------|
| `PIIModeRedact` | Reemplaza PII con `[REDACTED]` |
| `PIIModeMask` | Enmascara parcialmente (ej: `***@email.com`) |
| `PIIModeHash` | Reemplaza con hash del valor |

Ver [GUARDRAILS.md](GUARDRAILS.md) para documentación completa y el ejemplo
[14-guardrails](../examples/14-guardrails/README.md).

## Errors

Errores tipados con contexto enriquecido.

### KairosError

```go
import "github.com/jllopis/kairos/pkg/errors"

type KairosError struct {
    Code        ErrorCode
    Message     string
    Err         error           // Error original (unwrap)
    Context     map[string]any  // Contexto adicional
    Recoverable bool
    Attributes  map[string]string
}
```

### Códigos de error

| Código | Constante | Descripción |
|--------|-----------|-------------|
| `llm_error` | `CodeLLM` | Error del proveedor LLM |
| `tool_error` | `CodeTool` | Error en ejecución de tool |
| `memory_error` | `CodeMemory` | Error de almacenamiento |
| `policy_error` | `CodePolicy` | Acción denegada por política |
| `timeout` | `CodeTimeout` | Timeout excedido |
| `config_error` | `CodeConfig` | Error de configuración |
| `mcp_error` | `CodeMCP` | Error de protocolo MCP |
| `a2a_error` | `CodeA2A` | Error de protocolo A2A |
| `planner_error` | `CodePlanner` | Error del planner |
| `internal_error` | `CodeInternal` | Error interno |

### Uso

```go
// Crear error con contexto
err := errors.New(errors.CodeTool, "tool execution failed", originalErr).
    WithContext("tool_name", "search").
    WithAttribute("severity", "high").
    WithRecoverable(true)

// Comprobar si es un KairosError
var ke *errors.KairosError
if errors.AsKairosError(err, &ke) {
    fmt.Println(ke.Code, ke.Recoverable)
}
```

Ver [ERROR_HANDLING.md](ERROR_HANDLING.md) para patrones completos y el ejemplo
[09-error-handling](../examples/09-error-handling/README.md).

## Resilience

Patrones de resiliencia para operaciones que pueden fallar.

### Retry

```go
import "github.com/jllopis/kairos/pkg/resilience"

cfg := resilience.RetryConfig{}.
    WithMaxAttempts(3).
    WithInitialDelay(100 * time.Millisecond).
    WithMaxDelay(5 * time.Second).
    WithIsRecoverable(func(err error) bool {
        return errors.IsRecoverable(err)
    })

// Sin resultado
err := cfg.Do(ctx, func() error {
    return riskyOperation()
})

// Con resultado
result, err := cfg.DoWithResult(ctx, func() (string, error) {
    return fetchData()
})
```

### Circuit Breaker

```go
cb := resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
    FailureThreshold: 5,
    SuccessThreshold: 2,
    Timeout:          30 * time.Second,
})

result, err := cb.Call(func() (any, error) {
    return callExternalService()
})

// Estados: StateClosed, StateOpen, StateHalfOpen
fmt.Println(cb.State())
```

### Timeout

```go
// Sin resultado
err := resilience.WithTimeout(ctx, 5*time.Second, func(ctx context.Context) error {
    return longOperation(ctx)
})

// Con resultado
result, err := resilience.WithTimeoutResult(ctx, 5*time.Second,
    func(ctx context.Context) (string, error) {
        return longComputation(ctx)
    },
)
```

### Fallback

```go
// Función como fallback
fb := resilience.FallbackFunc(func(ctx context.Context, err error) (any, error) {
    return "default value", nil
})

// Valor estático
fb := resilience.NewStaticFallback("default value")

// Caché del último valor bueno
fb := resilience.NewCachedFallback()
fb.Update("last good value")

// Cadena de fallbacks (intenta cada uno en orden)
fb := resilience.NewChainedFallback(fallback1, fallback2, fallback3)

// Degradación gradual (cambia a fallback tras N errores)
gd := resilience.NewGracefulDegradation(primaryFunc, fallbackStrategy, 5)
result, err := gd.Execute(ctx)
```

Ver [INTEGRATION_GUIDE.md](INTEGRATION_GUIDE.md) para patrones de integración y el ejemplo
[10-resilience-patterns](../examples/10-resilience-patterns/README.md).

## Telemetry

Observabilidad nativa con OpenTelemetry.

### Inicialización

```go
import "github.com/jllopis/kairos/pkg/telemetry"

// Init básico (lee config de archivo/env)
shutdown, err := telemetry.Init(ctx)
defer shutdown(ctx)

// Init con config explícita
shutdown, err := telemetry.InitWithConfig(ctx, telemetry.Config{
    ServiceName: "my-agent",
    Exporter:    "otlp",                // "stdout", "otlp", "none"
    Endpoint:    "localhost:4317",
    Insecure:    true,
    Timeout:     30 * time.Second,
})
defer shutdown(ctx)
```

### Error Metrics

```go
metrics := telemetry.NewErrorMetrics(meter)

// Registrar error
metrics.RecordError(ctx, kairosErr)

// Registrar recuperación
metrics.RecordRecovery(ctx, kairosErr)

// Actualizar estado de salud
metrics.UpdateHealthStatus(ctx, core.HealthStatusHealthy)

// Actualizar estado de circuit breaker
metrics.UpdateCircuitBreakerState(ctx, "service-x", "open")
```

### Logging con trace context

```go
// Configura slog para inyectar trace_id y span_id automáticamente
telemetry.ConfigureSlog()

// Todos los logs incluyen trace context
slog.Info("processing request", "agent", agentID)
// Output: {"msg":"processing request","agent":"demo","trace_id":"abc123","span_id":"def456"}
```

### Atributos semánticos

Los atributos siguen convenciones OpenTelemetry:

```go
// En spans y métricas
span.SetAttributes(
    telemetry.AgentID("demo-agent"),
    telemetry.AgentRole("analyst"),
    telemetry.SessionID("user-123"),
    telemetry.ToolName("search"),
    telemetry.LLMModel("gpt-4"),
    telemetry.LLMTokensIn(150),
    telemetry.LLMTokensOut(200),
)
```

Ver [OBSERVABILITY.md](OBSERVABILITY.md) para dashboards y alertas, y el ejemplo
[11-observability](../examples/11-observability/README.md).

## Events

Sistema de eventos semánticos para streaming y auditoría.

### Interface

```go
type EventEmitter interface {
    Emit(ctx context.Context, event Event)
}

type Event struct {
    Type      EventType
    AgentID   string
    Data      map[string]any
    Timestamp time.Time
}
```

### Tipos de eventos

| Tipo | Descripción |
|------|-------------|
| `EventTypeThinking` | El agente está procesando |
| `EventTypeToolCall` | Llamada a tool |
| `EventTypeToolResult` | Resultado de tool |
| `EventTypeFinalAnswer` | Respuesta final |
| `EventTypeError` | Error durante ejecución |

### Uso

```go
emitter := myCustomEmitter{} // implementa core.EventEmitter

a, _ := agent.New("demo-agent", llmProvider,
    agent.WithEventEmitter(emitter),
)
```

`NoopEventEmitter` está disponible para desactivar eventos.

Ver [EVENT_TAXONOMY.md](EVENT_TAXONOMY.md) para la taxonomía completa.

## Health Checks

Sistema de health checks para monitorización.

### Interfaces

```go
type HealthChecker interface {
    Check(ctx context.Context) HealthResult
}

type HealthCheckProvider interface {
    RegisterChecker(name string, checker HealthChecker)
    CheckAll(ctx context.Context) ([]HealthResult, HealthStatus)
    Check(ctx context.Context, name string) (HealthResult, error)
}
```

### Estados

| Estado | Constante | Valor |
|--------|-----------|-------|
| Down | `HealthStatusDown` | 0 |
| Degraded | `HealthStatusDegraded` | 1 |
| Healthy | `HealthStatusHealthy` | 2 |

### Uso

```go
provider := core.NewDefaultHealthCheckProvider(
    core.WithCacheTTL(30 * time.Second),
)

provider.RegisterChecker("llm", agent.NewLLMHealthChecker(llmProvider))
provider.RegisterChecker("memory", agent.NewMemoryHealthChecker(mem))
provider.RegisterChecker("mcp", agent.NewMCPHealthChecker(mcpClient))

results, status := provider.CheckAll(ctx)
```

Checkers incluidos en `pkg/agent`:
- `AgentHealthChecker`: estado general del agente
- `LLMHealthChecker`: conectividad con el LLM provider
- `MemoryHealthChecker`: estado del backend de memoria
- `MCPHealthChecker`: conectividad con servidores MCP

## Discovery

Auto-descubrimiento y registro de agentes.

### Resolver

```go
import "github.com/jllopis/kairos/pkg/discovery"

resolver := discovery.NewResolver(
    discovery.NewConfigProvider(configEndpoints),
    discovery.NewWellKnownProvider("https://myhost.com"),
    discovery.NewRegistryProvider("https://registry.example.com"),
)

endpoints, err := resolver.Resolve(ctx, "agent-name")
```

### Providers de discovery

| Provider | Descripción |
|----------|-------------|
| `ConfigProvider` | Endpoints estáticos desde configuración |
| `WellKnownProvider` | Descubrimiento via `.well-known/agent.json` |
| `RegistryProvider` | Registro dinámico con API HTTP |

### Auto-registro

```go
stop, err := discovery.StartAutoRegister(ctx, discovery.AutoRegisterConfig{
    RegistryURL: "https://registry.example.com",
    AgentCard:   myAgentCard,
    Interval:    60 * time.Second,
})
defer stop()
```

## Governance y AGENTS.md

### Carga automática de AGENTS.md

Si existe `AGENTS.md`, se añade al prompt de sistema al crear el agente.

```go
doc, _ := governance.LoadAGENTS(".")
agent.New("demo-agent", llmProvider,
  agent.WithAGENTSInstructions(doc),
)
```

### Policy Engine

```go
engine := governance.NewRuleBasedEngine(governance.RuleSet{
    Rules: []governance.Rule{
        {Scope: "tool:*", Action: governance.ActionAllow},
        {Scope: "tool:dangerous_tool", Action: governance.ActionDeny},
        {Scope: "delegate:*", Action: governance.ActionRequireApproval},
    },
})

a, _ := agent.New("governed-agent", llmProvider,
    agent.WithPolicyEngine(engine),
)
```

Ver [governance-usage.md](governance-usage.md) y [governance-hitl.md](governance-hitl.md).

## Configuration

```go
cfg, err := config.Load("./.kairos/settings.json")
```

Para overrides por CLI, ver [CONFIGURATION.md](CONFIGURATION.md).

## Skills (AgentSkills)

Un skill se define con un `SKILL.md` en un directorio con el mismo nombre:

```
skills/
  pdf-processing/
    SKILL.md
```

Carga desde el agente:

```go
agent.New("demo-agent", llmProvider,
  agent.WithSkillsFromDir("./skills"),
)
```

El frontmatter usa `name`, `description`, `license`, `compatibility`,
`metadata` y `allowed-tools`.

Ver [Skills.md](Skills.md) para la especificación completa.
