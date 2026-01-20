# Referencia de API (curada)

Esta referencia cubre las APIs públicas más usadas. No intenta documentar todos
los paquetes, sino lo esencial para construir agentes y flujos con Kairos.

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

## Task (core)

Crear y propagar una Task:

```go
task := core.NewTask("Resumir ventas", "orchestrator")
ctx = core.WithTask(ctx, task)
```

Cuando el agente ejecuta con una Task en contexto, actualiza estado y resultado
automáticamente. Ver `docs/TASKS.md` para el detalle.

## Memory

### Semantic Memory (Vector)

Ejemplo con Qdrant + Ollama:

```go
store, _ := qdrant.New("localhost:6334")
embedder := ollama.NewEmbedder("http://localhost:11434", "nomic-embed-text")
mem, _ := memory.NewVectorMemory(context.Background(), store, embedder, "kairos")
_ = mem.Initialize(context.Background())

_ = mem.Store(ctx, "Mi color favorito es azul.")
matches, _ := mem.Retrieve(ctx, "color favorito")
```

### Conversation Memory (Chat History)

Para conversaciones multi-turno:

```go
// Crear memoria con estrategia de ventana
convMem := memory.NewInMemoryConversation(memory.ConversationConfig{
    TruncationStrategy: memory.NewWindowStrategy(20, true),
})

// Usar con el agente
a, _ := agent.New("chat-agent", llmProvider,
    agent.WithConversationMemory(convMem),
)

// Ejecutar con sessionID
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

Ver `docs/CONVERSATION_MEMORY.md` para documentación completa.

## Governance y AGENTS.md

Carga automática:

- Si existe `AGENTS.md`, se añade al prompt de sistema al crear el agente.

Carga manual:

```go
doc, _ := governance.LoadAGENTS(".")
agent.New("demo-agent", llmProvider,
  agent.WithAGENTSInstructions(doc),
)
```

## MCP

Cliente MCP por stdio:

```go
client, _ := mcp.NewClientWithStdioProtocol("node", []string{serverPath}, "2024-11-05")
agent.New("demo-agent", llmProvider,
  agent.WithMCPClients(client),
)
```

## A2A (server)

Handler mínimo:

```go
handler := &server.SimpleHandler{
  Store:    server.NewMemoryTaskStore(),
  Executor: myExecutor{},
  Card:     myAgentCard(),
  PushCfgs: server.NewMemoryPushConfigStore(),
  ApprovalStore: server.NewMemoryApprovalStore(),
}
```

Para bindings, ver `docs/protocols/A2A/topics/bindings.md`.

## LLM Provider

El agente requiere un `llm.Provider` con el método `Chat`. Para pruebas, puedes
usar `llm.MockProvider` o `llm.ScriptedMockProvider`.

```go
llmProvider := &llm.MockProvider{Response: "ok"}
a, _ := agent.New("demo-agent", llmProvider)
```

## Configuración

Carga un `settings.json` con:

```go
cfg, err := config.Load("./.kairos/settings.json")
if err != nil {
  // handle error
}
```

Para overrides por CLI, ver `docs/CONFIGURATION.md`.
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
