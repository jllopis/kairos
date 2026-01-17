# Conectores - Kairos

Los **Conectores** transforman especificaciones externas (OpenAPI, GraphQL, etc.) en `[]llm.Tool` que cualquier LLM provider puede usar.

## Arquitectura

```
┌─────────────────────────────────────────────────────────────┐
│                         Agent                                │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│   ┌──────────────────┐         ┌──────────────────────────┐ │
│   │    Providers     │         │       Connectors         │ │
│   │    (LLMs)        │         │       (Tools)            │ │
│   ├──────────────────┤         ├──────────────────────────┤ │
│   │ OpenAI      ✅   │         │ OpenAPIConnector    ✅   │ │
│   │ Anthropic   ✅   │         │ MCPConnector        ✅   │ │
│   │ Gemini      ✅   │         │ GraphQLConnector    🔜   │ │
│   │ Qwen        ✅   │         │ GRPCConnector       🔜   │ │
│   │ Ollama      ✅   │         │ SQLConnector        🔜   │ │
│   └──────────────────┘         └──────────────────────────┘ │
│            │                              │                  │
│            │       ┌──────────────┐       │                  │
│            └──────►│  llm.Tool[]  │◄──────┘                  │
│                    │ (formato     │                          │
│                    │  común)      │                          │
│                    └──────────────┘                          │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Principio de diseño

| Componente | Responsabilidad |
|------------|-----------------|
| **Providers** | Comunicación con LLMs (OpenAI, Claude, Gemini...) |
| **Connectors** | Generación de `[]llm.Tool` desde specs externos |
| **Tools** | Formato común que todos los providers entienden |

Esta separación permite:
- ✅ Usar **cualquier conector** con **cualquier provider**
- ✅ Añadir nuevos conectores sin modificar providers
- ✅ Añadir nuevos providers sin modificar conectores

## Conectores disponibles

| Conector | Spec de entrada | Tools generados | Estado |
|----------|-----------------|-----------------|--------|
| `OpenAPIConnector` | OpenAPI 3.x, Swagger 2.0 | REST endpoints | ✅ Implementado |
| `MCPConnector` | MCP protocol | MCP tools | ✅ Implementado |
| `GraphQLConnector` | Schema introspection | Queries/Mutations | ✅ Implementado |
| `GRPCConnector` | `.proto` files | RPC methods | 🔜 Planificado |
| `SQLConnector` | Database schema | CRUD operations | 🔜 Planificado |

## OpenAPIConnector

Convierte especificaciones OpenAPI/Swagger en tools ejecutables automáticamente.

### Características

- **Parsea** OpenAPI 3.x y Swagger 2.0 (YAML y JSON)
- **Genera** un `llm.Tool` por cada operación (GET, POST, PUT, DELETE, PATCH)
- **Extrae** parámetros de path, query, header y request body
- **Convierte** schemas a JSON Schema para validación del LLM
- **Ejecuta** llamadas HTTP reales con autenticación configurada

### Uso básico

```go
import "github.com/jllopis/kairos/pkg/connectors"

// Crear conector desde URL o archivo local
connector, err := connectors.NewOpenAPIConnector(
    "https://api.example.com/openapi.yaml",
)

// Obtener tools generados
tools := connector.Tools()  // []llm.Tool

// Usar con cualquier provider
agent := kairos.NewAgent(
    kairos.WithProvider(openaiProvider),
    kairos.WithTools(tools...),
)
```

### Autenticación

```go
// API Key en header personalizado
connector, _ := connectors.NewOpenAPIConnector(spec,
    connectors.WithAPIKey("sk-xxx", "X-API-Key"),
)

// Bearer token
connector, _ := connectors.NewOpenAPIConnector(spec,
    connectors.WithBearerToken(os.Getenv("API_TOKEN")),
)

// HTTP Basic Auth
connector, _ := connectors.NewOpenAPIConnector(spec,
    connectors.WithBasicAuth("user", "password"),
)
```

### Ejecución manual de tools

Si necesitas ejecutar tools fuera del agent loop:

```go
result, err := connector.Execute(ctx, "createPet", map[string]any{
    "name": "Buddy",
    "type": "dog",
})
```

### Ejemplo: Pet Store API

```go
// spec: https://petstore3.swagger.io/api/v3/openapi.json
connector, _ := connectors.NewOpenAPIConnector(
    "https://petstore3.swagger.io/api/v3/openapi.json",
)

tools := connector.Tools()
// Genera tools como:
// - addPet (POST /pet)
// - updatePet (PUT /pet)
// - findPetsByStatus (GET /pet/findByStatus)
// - getPetById (GET /pet/{petId})
// - deletePet (DELETE /pet/{petId})
// ...
```

Ver `examples/17-openapi-connector/` para un ejemplo completo.

## GraphQLConnector

Convierte esquemas GraphQL en tools ejecutables mediante introspección.

### Características

- **Introspección automática**: Descubre queries y mutations del endpoint
- **Genera** un `llm.Tool` por cada query y mutation
- **Mapea** argumentos GraphQL a JSON Schema
- **Ejecuta** queries/mutations con los argumentos proporcionados
- **Soporta** autenticación (Bearer, API Key, headers personalizados)

### Uso básico

```go
import "github.com/jllopis/kairos/pkg/connectors"

// Crear conector con introspección automática
connector, err := connectors.NewGraphQLConnector(
    "https://api.example.com/graphql",
)

// Obtener tools generados
tools := connector.Tools()  // []llm.Tool

// Usar con cualquier provider
agent := kairos.NewAgent(
    kairos.WithProvider(openaiProvider),
    kairos.WithTools(tools...),
)
```

### Autenticación

```go
// Bearer token
connector, _ := connectors.NewGraphQLConnector(endpoint,
    connectors.WithGraphQLBearerToken(os.Getenv("GITHUB_TOKEN")),
)

// API Key
connector, _ := connectors.NewGraphQLConnector(endpoint,
    connectors.WithGraphQLAPIKey("my-key", "X-API-Key"),
)

// Header personalizado
connector, _ := connectors.NewGraphQLConnector(endpoint,
    connectors.WithGraphQLHeader("X-Custom-Header", "value"),
)
```

### Ejecución de queries

```go
// El conector detecta automáticamente si es query o mutation
result, err := connector.Execute(ctx, "user", map[string]interface{}{
    "id": "123",
})

// Mutations
result, err := connector.Execute(ctx, "createUser", map[string]interface{}{
    "name":  "John Doe",
    "email": "john@example.com",
})
```

### Prefijo de tools

Para evitar colisiones de nombres al combinar múltiples conectores:

```go
connector, _ := connectors.NewGraphQLConnector(endpoint,
    connectors.WithGraphQLToolPrefix("github"),
)
// Genera: github_user, github_repository, etc.
```

Ver `examples/19-graphql-connector/` para un ejemplo completo.

## MCPConnector

El conector MCP ya está implementado en `pkg/mcp/` y permite:

- Conectar con servidores MCP (stdio, HTTP, WebSocket)
- Obtener tools via `ListTools()`
- Ejecutar tools via `CallTool()`

```go
import "github.com/jllopis/kairos/pkg/mcp"

client, _ := mcp.NewStdioClient(mcp.StdioConfig{
    Command: "npx",
    Args:    []string{"-y", "@modelcontextprotocol/server-filesystem", "/tmp"},
})

tools, _ := client.ListTools(ctx)
result, _ := client.CallTool(ctx, "read_file", map[string]any{"path": "/tmp/test.txt"})
```

## Conectores futuros

### GRPCConnector (planificado)

```go
// Futuro API
connector, _ := connectors.NewGRPCConnector(
    "localhost:50051",
    connectors.WithProtoFiles("api/v1/*.proto"),
)

tools := connector.Tools()
// Genera tools desde métodos RPC
```

### SQLConnector (planificado)

```go
// Futuro API
connector, _ := connectors.NewSQLConnector(
    "postgres://user:pass@localhost/db",
    connectors.WithTables("users", "orders", "products"),
)

tools := connector.Tools()
// Genera tools: listUsers, getUser, createUser, updateUser, deleteUser...
```

## Implementar un conector personalizado

Para crear un conector nuevo, implementa la interfaz implícita:

```go
type MyConnector struct {
    // ...
}

// Tools genera []llm.Tool desde tu especificación
func (c *MyConnector) Tools() []llm.Tool {
    return []llm.Tool{
        {
            Type: "function",
            Function: llm.FunctionDef{
                Name:        "myTool",
                Description: "Does something useful",
                Parameters: map[string]any{
                    "type": "object",
                    "properties": map[string]any{
                        "input": map[string]any{
                            "type":        "string",
                            "description": "Input value",
                        },
                    },
                    "required": []string{"input"},
                },
            },
        },
    }
}

// Execute invoca el tool con los argumentos dados
func (c *MyConnector) Execute(ctx context.Context, name string, args map[string]any) (any, error) {
    switch name {
    case "myTool":
        return c.doMyTool(args["input"].(string))
    default:
        return nil, fmt.Errorf("unknown tool: %s", name)
    }
}
```

## Integración con el Agent

Los conectores se integran con el agent de dos formas:

### 1. Tools estáticos (al crear el agent)

```go
connector, _ := connectors.NewOpenAPIConnector(spec)

agent := kairos.NewAgent(
    kairos.WithProvider(provider),
    kairos.WithTools(connector.Tools()...),
)
```

### 2. Tool execution en el loop

El agent loop detecta tool calls del LLM y las ejecuta:

```go
// En el agent loop (simplificado)
for _, toolCall := range response.ToolCalls {
    result, err := connector.Execute(ctx, toolCall.Function.Name, toolCall.Function.Arguments)
    // ... añade resultado al contexto
}
```

## Recursos

- **Código**: `pkg/connectors/`
- **Tests**: `pkg/connectors/openapi_test.go`
- **Ejemplos**: `examples/17-openapi-connector/`
- **Documentación relacionada**: [PROVIDERS.md](PROVIDERS.md), [ARCHITECTURE.md](ARCHITECTURE.md)
