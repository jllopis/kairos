# Example 19: GraphQL Connector

Este ejemplo demuestra cómo usar el conector GraphQL para generar automáticamente tools desde un schema GraphQL.

## Cómo funciona

1. **Introspección**: El conector conecta al endpoint GraphQL y ejecuta una query de introspección
2. **Generación**: Cada query y mutation se convierte en un `llm.Tool`
3. **Ejecución**: Cuando el LLM invoca un tool, el conector construye y ejecuta la query GraphQL

## Ejecutar

```bash
# Usa la API pública de Countries (por defecto)
go run main.go

# O especifica tu propio endpoint
GRAPHQL_ENDPOINT="https://api.example.com/graphql" go run main.go
```

## Con autenticación

```go
connector, err := connectors.NewGraphQLConnector(
    "https://api.github.com/graphql",
    connectors.WithGraphQLBearerToken(os.Getenv("GITHUB_TOKEN")),
)
```

## Salida esperada

```
🔗 GraphQL Connector Example
============================
Endpoint: https://countries.trevorblades.com/graphql

📡 Performing introspection...

✅ Generated 5 tools from schema:

   📦 continent
      Args: code
   📦 continents
      Args: filter
   📦 countries
      Args: filter
   📦 country
      Args: code
   📦 language
      Args: code
```

## Integración con Kairos

```go
// Crear conector
connector, _ := connectors.NewGraphQLConnector(endpoint,
    connectors.WithGraphQLBearerToken(token),
)

// Usar tools con cualquier provider
agent := kairos.NewAgent(
    kairos.WithProvider(openaiProvider),
    kairos.WithTools(connector.Tools()...),
)

// El agente puede ahora hacer queries GraphQL
result, _ := agent.Run(ctx, "List all European countries")
```
