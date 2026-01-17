# Example 17: OpenAPI Connector

Este ejemplo demuestra cómo usar el conector OpenAPI para convertir automáticamente cualquier API REST en tools compatibles con LLM.

## Características

- **Conversión automática**: Parsea specs OpenAPI 3.x (YAML o JSON)
- **Generación de tools**: Crea `llm.Tool` para cada operación
- **Autenticación**: Soporte para API Key, Bearer token y Basic auth
- **Ejecución integrada**: Ejecuta las llamadas HTTP automáticamente

## Uso básico

```go
import "github.com/jllopis/kairos/pkg/connectors"

// Desde archivo
connector, err := connectors.NewFromFile("openapi.yaml")

// Desde URL
connector, err := connectors.NewFromURL("https://api.example.com/openapi.json")

// Desde bytes
connector, err := connectors.NewFromBytes(specData)
```

## Opciones de configuración

```go
connector, err := connectors.NewFromFile("spec.yaml",
    // Override base URL
    connectors.WithBaseURL("https://api.example.com"),
    
    // API Key auth
    connectors.WithAPIKey("your-api-key", "X-API-Key"),
    
    // Bearer token auth
    connectors.WithBearerToken("your-token"),
    
    // Basic auth
    connectors.WithBasicAuth("user", "password"),
    
    // Custom HTTP client
    connectors.WithHTTPClient(customClient),
)
```

## Obtener tools generados

```go
// Obtener todos los tools para pasarlos al agente
tools := connector.Tools()

// Los tools son compatibles con llm.Tool
agent := agent.New(
    agent.WithProvider(provider),
    agent.WithTools(tools...),
)
```

## Ejecutar tools

```go
// Con argumentos como map
result, err := connector.Execute(ctx, "listPets", map[string]interface{}{
    "limit":   10,
    "species": "dog",
})

// Con argumentos JSON (útil para tool calls del LLM)
result, err := connector.ExecuteJSON(ctx, "getPet", `{"id": "123"}`)
```

## Ejecutar el ejemplo

```bash
cd examples/17-openapi-connector
go run .
```

## Salida esperada

```
=== OpenAPI Connector Demo ===

Generated Tools:
----------------
• listPets: List all pets in the store
  Parameters: {
    "type": "object",
    "properties": {
      "limit": {"type": "integer", "description": "Maximum number of pets to return"},
      "species": {"type": "string", "enum": ["dog", "cat", "bird", "fish"]}
    }
  }
...

Executing Tools:
----------------

1. Listing all pets...
   Result: [{"age":5,"id":"1","name":"Max","species":"dog"},...]

2. Creating a new pet...
   Result: {"age":3,"id":"4","name":"Buddy","species":"dog"}

✓ Demo completed!
```

## Integración con agentes

```go
// El conector puede integrarse con un agente Kairos
connector, _ := connectors.NewFromFile("api-spec.yaml")

// Crear skill handler que delega en el conector
handler := func(ctx context.Context, call llm.ToolCall) (string, error) {
    return connector.ExecuteJSON(ctx, call.Function.Name, call.Function.Arguments)
}

// Registrar tools en el agente
for _, tool := range connector.Tools() {
    agent.RegisterTool(tool, handler)
}
```

## Specs soportados

- OpenAPI 3.0.x
- OpenAPI 3.1.x
- Formato YAML y JSON
- Parámetros: path, query, header
- Request body: application/json
- Respuestas: cualquier content-type (devuelve como string)
