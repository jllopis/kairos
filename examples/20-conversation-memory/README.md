# 20 - Conversation Memory

Agente con memoria de conversación para interacciones multi-turno.

## Qué aprenderás

- Configurar `ConversationMemory` para mantener historial de chat
- Usar `sessionID` para identificar conversaciones
- Estrategias de truncado (ventana, tokens, resumen)
- Diferencia entre memoria semántica y memoria de conversación

## Requisitos

- Ollama instalado y corriendo (`ollama serve`)
- Modelo disponible (por defecto `llama3.2`):
  ```bash
  ollama pull llama3.2
  ```

## Ejecutar

```bash
cd examples/20-conversation-memory
go run .

# O con un modelo diferente:
OLLAMA_MODEL=mistral go run .
```

## Variables de entorno

| Variable | Default | Descripción |
|----------|---------|-------------|
| `OLLAMA_URL` | `http://localhost:11434` | URL del servidor Ollama |
| `OLLAMA_MODEL` | `llama3.2` | Modelo a usar |

## Salida esperada

```
=== Conversation Memory Demo ===
Este ejemplo demuestra cómo un agente mantiene el contexto
entre múltiples interacciones usando ConversationMemory.

--- Turno 1 ---
Usuario: Hola, me llamo Juan
Agente: ¡Hola Juan! Encantado de conocerte. ¿En qué puedo ayudarte hoy?

--- Turno 2 ---
Usuario: ¿Cómo me llamo?
Agente: Te llamas Juan, me lo dijiste hace un momento.

--- Turno 3 ---
Usuario: Quiero aprender a usar Kairos, ¿por dónde empiezo?
Agente: Claro Juan, te recomiendo empezar con el ejemplo 01-hello-agent...

--- Turno 4 ---
Usuario: ¿De qué hemos hablado?
Agente: Hemos hablado sobre tu nombre (Juan) y te he recomendado empezar...

=== Historial de la conversación ===
[Usuario] Hola, me llamo Juan
[Agente] ¡Hola Juan! Encantado de conocerte...
...

Total de mensajes almacenados: 8
```

## Conceptos clave

### ConversationMemory vs Memory

| Aspecto | ConversationMemory | Memory (Vector) |
|---------|-------------------|-----------------|
| Propósito | Historial de chat ordenado | Conocimiento semántico |
| Estructura | Secuencial por tiempo | Embeddings por similitud |
| Consulta | Por sessionID | Por relevancia |
| Uso | Conversaciones multi-turno | RAG, contexto externo |

### SessionID

Cada conversación se identifica con un `sessionID`:

```go
// Crear contexto con sessionID
ctx := core.WithSessionID(context.Background(), "user-123-session")

// El agente guarda/recupera mensajes de esta sesión
a.Run(ctx, "Hola")
a.Run(ctx, "¿Qué te dije antes?")  // Recuerda el "Hola"
```

### Estrategias de truncado

El historial puede crecer. Las estrategias lo limitan:

```go
// Ventana: últimos N mensajes
strategy := memory.NewWindowStrategy(20, true)

// Tokens: máximo de tokens estimados
strategy := memory.NewTokenStrategy(4000, true)

// Resumen: resumir mensajes antiguos
strategy := memory.NewSummarizationStrategy(20, 10, summarizeFunc)
```

## Código clave

```go
// 1. Crear memoria de conversación
convMem := memory.NewInMemoryConversation(memory.ConversationConfig{
    TruncationStrategy: memory.NewWindowStrategy(20, true),
})

// 2. Crear agente con memoria
a, _ := agent.New("chat-agent", llmProvider,
    agent.WithConversationMemory(convMem),
)

// 3. Usar sessionID para identificar la conversación
ctx := core.WithSessionID(context.Background(), "session-123")

// 4. Cada Run guarda y recupera el historial automáticamente
a.Run(ctx, "Mensaje 1")
a.Run(ctx, "Mensaje 2")  // Tiene contexto del mensaje 1
```

## Backends disponibles

### In-Memory (este ejemplo)

```go
convMem := memory.NewInMemoryConversation(config)
```

### File-based

```go
convMem, _ := memory.NewFileConversation("./conversations", config)
```

### PostgreSQL

```go
db, _ := sql.Open("postgres", connString)
convMem, _ := memory.NewPostgresConversation(memory.PostgresConfig{
    DB: db,
})
convMem.Initialize(ctx)
```

## Múltiples sesiones

Un mismo agente puede manejar múltiples conversaciones simultáneas:

```go
// Usuario A
ctxA := core.WithSessionID(ctx, "user-a-session")
a.Run(ctxA, "Soy Alice")

// Usuario B (conversación separada)
ctxB := core.WithSessionID(ctx, "user-b-session")
a.Run(ctxB, "Soy Bob")

// Cada sesión mantiene su propio historial
```

## Limpiar historial

```go
// Limpiar una sesión específica
convMem.Clear(ctx, "session-123")

// Eliminar mensajes antiguos (más de 1 hora)
convMem.DeleteOldMessages(ctx, "session-123", time.Hour)
```

## Siguiente paso

→ Revisa [docs/CONVERSATION_MEMORY.md](/docs/CONVERSATION_MEMORY.md) para documentación completa.
