# Conectores - Kairos

Los **Conectores** transforman especificaciones externas (OpenAPI, GraphQL, etc.) en `[]core.Tool` listos para el agent. Cada tool expone su `llm.Tool` via `ToolDefinition()`.

## Cambio breaking (Tools y Execute)

- `Tools()` ahora devuelve `[]core.Tool` (antes `[]llm.Tool`).
- Usa `tool.ToolDefinition()` si necesitas el schema `llm.Tool`.
- `OpenAPIConnector.Execute` y `ExecuteJSON` ahora devuelven `any`.

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
│            │       ┌───────────────┐      │                  │
│            └──────►│  core.Tool[]  │◄─────┘                  │
│                    │ (ToolDefinition)                       │
│                    └───────────────┘                         │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Principio de diseño

| Componente | Responsabilidad |
|------------|-----------------|
| **Providers** | Comunicación con LLMs (OpenAI, Claude, Gemini...) |
| **Connectors** | Generación de `[]core.Tool` desde specs externos |
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
| `GRPCConnector` | Server reflection | RPC methods | ✅ Implementado |
| `SQLConnector` | Database schema | CRUD operations | ✅ Implementado |

## OpenAPIConnector

Convierte especificaciones OpenAPI/Swagger en tools ejecutables automáticamente.

### Características

- **Parsea** OpenAPI 3.x y Swagger 2.0 (YAML y JSON)
- **Genera** un `core.Tool` por cada operación (GET, POST, PUT, DELETE, PATCH)
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
tools := connector.Tools()  // []core.Tool

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
- **Genera** un `core.Tool` por cada query y mutation
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
tools := connector.Tools()  // []core.Tool

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

## GRPCConnector

Convierte servicios gRPC en tools mediante reflection del servidor.

### Características

- **Server reflection**: Descubre servicios y métodos automáticamente
- **Genera** un `core.Tool` por cada método RPC (excepto streaming)
- **Mapea** tipos protobuf a JSON Schema
- **Ejecuta** llamadas gRPC dinámicamente
- **Soporta** conexiones seguras e inseguras

### Uso básico

```go
import "github.com/jllopis/kairos/pkg/connectors"

// Crear conector con reflection (requiere que el servidor tenga reflection habilitado)
connector, err := connectors.NewGRPCConnector(
    "localhost:50051",
    connectors.WithGRPCInsecure(), // Para desarrollo
)
defer connector.Close()

// Obtener tools generados
tools := connector.Tools()  // []core.Tool

// Usar con cualquier provider
agent := kairos.NewAgent(
    kairos.WithProvider(openaiProvider),
    kairos.WithTools(tools...),
)
```

### Ejecución de métodos

```go
// Los nombres de tools siguen el formato: service_name_method_name (snake_case)
result, err := connector.Execute(ctx, "user_service_get_user", map[string]interface{}{
    "id": "123",
})
```

### Opciones

```go
// Prefijo para evitar colisiones
connector, _ := connectors.NewGRPCConnector(target,
    connectors.WithGRPCToolPrefix("myapi"),
    connectors.WithGRPCInsecure(),
)

// Con opciones de dial personalizadas
connector, _ := connectors.NewGRPCConnector(target,
    connectors.WithGRPCDialOptions(
        grpc.WithTransportCredentials(creds),
    ),
)
```

### Requisitos

El servidor gRPC debe tener **reflection habilitado**:

```go
// En el servidor gRPC
import "google.golang.org/grpc/reflection"

s := grpc.NewServer()
reflection.Register(s)
```

## SQLConnector

Genera operaciones CRUD automáticamente desde un esquema de base de datos.

### Características

- **Introspección automática** del esquema (information_schema o PRAGMA)
- **Genera 5 tools por tabla**: list, get, create, update, delete
- **Mapea** tipos SQL a JSON Schema
- **Soporta** PostgreSQL, MySQL y SQLite
- **Modo read-only** opcional

### Uso básico

```go
import (
    "database/sql"
    "github.com/jllopis/kairos/pkg/connectors"
    _ "modernc.org/sqlite" // o "github.com/lib/pq" para Postgres
)

// Abrir conexión a la base de datos
db, _ := sql.Open("sqlite", "database.db")

// Crear conector con introspección
connector, err := connectors.NewSQLConnector(db, "sqlite")

// Obtener tools generados
tools := connector.Tools()
// Para una tabla "users" genera:
// - list_users    (SELECT con filtros, limit, offset)
// - get_users     (SELECT by primary key)
// - create_users  (INSERT)
// - update_users  (UPDATE)
// - delete_users  (DELETE)
```

### Ejecución de operaciones

```go
ctx := context.Background()

// Listar con filtros
result, _ := connector.Execute(ctx, "list_users", map[string]interface{}{
    "filters": map[string]interface{}{
        "status": "active",
    },
    "limit":    10,
    "offset":   0,
    "order_by": "created_at",
    "order_desc": true,
})

// Obtener uno por ID
result, _ := connector.Execute(ctx, "get_users", map[string]interface{}{
    "id": 123,
})

// Crear
result, _ := connector.Execute(ctx, "create_users", map[string]interface{}{
    "name":  "John Doe",
    "email": "john@example.com",
})

// Actualizar
result, _ := connector.Execute(ctx, "update_users", map[string]interface{}{
    "id":    123,
    "name":  "Jane Doe",
})

// Eliminar
result, _ := connector.Execute(ctx, "delete_users", map[string]interface{}{
    "id": 123,
})
```

### Opciones

```go
// Solo lectura (no genera create, update, delete)
connector, _ := connectors.NewSQLConnector(db, "postgres",
    connectors.WithSQLReadOnly(),
)

// Con prefijo
connector, _ := connectors.NewSQLConnector(db, "mysql",
    connectors.WithSQLToolPrefix("db"),
)
// Genera: db_list_users, db_get_users, etc.
```

### Drivers soportados

| Driver | Package | Ejemplo DSN |
|--------|---------|-------------|
| PostgreSQL | `github.com/lib/pq` | `postgres://user:pass@localhost/db` |
| MySQL | `github.com/go-sql-driver/mysql` | `user:pass@tcp(localhost:3306)/db` |
| SQLite | `modernc.org/sqlite` | `file.db` o `:memory:` |

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

## Implementar un conector personalizado

Para crear un conector nuevo, implementa la interfaz implícita:

```go
type MyConnector struct {
    // ...
}

// Tools genera []core.Tool desde tu especificación
func (c *MyConnector) Tools() []core.Tool {
    return []core.Tool{
        &MyTool{connector: c},
    }
}

type MyTool struct {
    connector *MyConnector
}

func (t *MyTool) Name() string {
    return "myTool"
}

func (t *MyTool) ToolDefinition() llm.Tool {
    return llm.Tool{
        Type: llm.ToolTypeFunction,
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
    }
}

func (t *MyTool) Call(ctx context.Context, input any) (any, error) {
    args := input.(map[string]any)
    return t.connector.Execute(ctx, "myTool", args)
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
    args := map[string]any{}
    _ = json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
    result, err := connector.Execute(ctx, toolCall.Function.Name, args)
    // ... añade resultado al contexto
}
```

## Recursos

- **Código**: `pkg/connectors/`
- **Tests**: `pkg/connectors/openapi_test.go`
- **Ejemplos**: `examples/17-openapi-connector/`
- **Documentación relacionada**: [PROVIDERS.md](PROVIDERS.md), [ARCHITECTURE.md](ARCHITECTURE.md)
