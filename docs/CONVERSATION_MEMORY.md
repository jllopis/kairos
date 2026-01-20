# Conversation Memory

Kairos soporta **memoria de conversación** para mantener el contexto entre múltiples interacciones con un agente. Esto permite conversaciones multi-turno donde el agente recuerda mensajes previos.

## Conceptos clave

- **SessionID**: Identificador único de una conversación. Múltiples usuarios pueden tener sesiones simultáneas.
- **ConversationMemory**: Interfaz para almacenar/recuperar historial de mensajes.
- **TruncationStrategy**: Estrategia para limitar el tamaño del historial (ventana, tokens, resumen).

## Uso básico

```go
import (
    "github.com/jllopis/kairos/pkg/agent"
    "github.com/jllopis/kairos/pkg/core"
    "github.com/jllopis/kairos/pkg/memory"
)

// Crear memoria de conversación in-memory
convMem := memory.NewInMemoryConversation(memory.ConversationConfig{
    TruncationStrategy: memory.NewWindowStrategy(20, true), // últimos 20 mensajes, mantener system
})

// Crear agente con memoria de conversación
a, err := agent.New("chat-agent", llmProvider,
    agent.WithConversationMemory(convMem),
)

// Ejecutar con sessionID
ctx := core.WithSessionID(context.Background(), "user-123-session")
resp1, _ := a.Run(ctx, "Hola, me llamo Juan")
resp2, _ := a.Run(ctx, "¿Cómo me llamo?")  // El agente recuerda "Juan"
```

## Backends disponibles

### In-Memory (desarrollo/testing)

```go
convMem := memory.NewInMemoryConversation(memory.ConversationConfig{})
```

- Rápido, sin dependencias
- Datos perdidos al reiniciar
- Ideal para desarrollo y tests

### File-based (persistencia simple)

```go
convMem, err := memory.NewFileConversation("./conversations", memory.ConversationConfig{})
```

- Persistencia en archivos JSON
- Un archivo por sesión
- Sin dependencias externas
- Adecuado para aplicaciones single-instance

### PostgreSQL (producción)

```go
import "database/sql"
import _ "github.com/lib/pq"

db, _ := sql.Open("postgres", "postgres://user:pass@localhost/kairos?sslmode=disable")

convMem, err := memory.NewPostgresConversation(memory.PostgresConfig{
    DB:        db,
    TableName: "conversation_messages",  // opcional, default: "conversation_messages"
})

// Crear tabla (primera vez)
convMem.Initialize(context.Background())
```

- Escalable y distribuido
- Múltiples instancias pueden compartir sesiones
- Consultas avanzadas y TTL

## Estrategias de truncado

El historial puede crecer indefinidamente. Las estrategias limitan su tamaño.

### WindowStrategy (ventana deslizante)

Mantiene los últimos N mensajes:

```go
// Últimos 10 mensajes, preservando system messages
strategy := memory.NewWindowStrategy(10, true)

convMem := memory.NewInMemoryConversation(memory.ConversationConfig{
    TruncationStrategy: strategy,
})
```

### TokenStrategy (presupuesto de tokens)

Limita por estimación de tokens:

```go
strategy := memory.NewTokenStrategy(4000, true) // ~4000 tokens, mantener system

// Opcional: contador de tokens personalizado
strategy.TokenCounter = func(msg memory.ConversationMessage) int {
    return tiktoken.Count(msg.Content) // usar tiktoken u otro
}
```

### SummarizationStrategy (resumen)

Resume mensajes antiguos usando el LLM:

```go
strategy := memory.NewSummarizationStrategy(
    20,  // max mensajes antes de resumir
    10,  // cuántos mensajes resumir a la vez
    func(ctx context.Context, msgs []memory.ConversationMessage) (string, error) {
        // Llamar al LLM para generar resumen
        prompt := "Resume esta conversación:\n"
        for _, m := range msgs {
            prompt += fmt.Sprintf("%s: %s\n", m.Role, m.Content)
        }
        return llmProvider.Summarize(ctx, prompt)
    },
)
```

## API de ConversationMemory

```go
type ConversationMemory interface {
    // Añadir mensaje a la conversación
    AppendMessage(ctx context.Context, sessionID string, msg ConversationMessage) error

    // Obtener todos los mensajes (aplica truncación si configurada)
    GetMessages(ctx context.Context, sessionID string) ([]ConversationMessage, error)

    // Obtener últimos N mensajes
    GetRecentMessages(ctx context.Context, sessionID string, limit int) ([]ConversationMessage, error)

    // Eliminar todos los mensajes de una sesión
    Clear(ctx context.Context, sessionID string) error

    // Eliminar mensajes antiguos
    DeleteOldMessages(ctx context.Context, sessionID string, olderThan time.Duration) error
}
```

## Estructura de mensaje

```go
type ConversationMessage struct {
    ID         string            // UUID generado automáticamente
    SessionID  string            // ID de la sesión
    Role       string            // "system", "user", "assistant", "tool"
    Content    string            // Contenido del mensaje
    ToolCallID string            // ID de tool call (si aplica)
    Metadata   map[string]string // Metadatos adicionales
    CreatedAt  time.Time         // Timestamp
}
```

## Gestión de sesiones

### SessionID automático

Si no se proporciona sessionID, se genera uno automáticamente:

```go
// Sin sessionID - se genera automáticamente
resp, _ := a.Run(context.Background(), "Hola")
```

### SessionID explícito

Para conversaciones persistentes:

```go
// Con sessionID explícito
ctx := core.WithSessionID(context.Background(), "user-abc-conv-1")
resp, _ := a.Run(ctx, "Hola")
```

### Limpiar una sesión

```go
convMem.Clear(ctx, "user-abc-conv-1")
```

### TTL de sesiones (PostgreSQL)

```go
// Eliminar sesiones inactivas por más de 24 horas
deleted, err := convMem.DeleteOldSessions(ctx, 24*time.Hour)
```

## Diferencia con Memory (semántica)

| Aspecto | ConversationMemory | Memory (Vector) |
|---------|-------------------|-----------------|
| Propósito | Historial de chat | Contexto semántico |
| Estructura | Secuencial, ordenado | Embeddings, similitud |
| Consulta | Por sessionID | Por relevancia semántica |
| Uso | Multi-turno | RAG, conocimiento |

Ambos pueden usarse simultáneamente:

```go
a, _ := agent.New("agent", llmProvider,
    agent.WithConversationMemory(convMem),  // Historial de chat
    agent.WithMemory(vectorMem),            // Conocimiento semántico
)
```

## Ejemplo completo

Ver `examples/conversation-agent` para un ejemplo completo con persistencia.
