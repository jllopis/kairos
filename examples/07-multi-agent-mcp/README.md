# 07 - Multi-Agent MCP

Agente que se comunica con servidores MCP remotos via HTTP.

## Qué aprenderás

- Conectar a servidores MCP remotos (HTTP)
- Descubrir herramientas de servicios externos
- Uso directo del cliente MCP
- Diferencia entre mock y LLM real

## Requisitos

1. Levantar un servidor MCP HTTP:

```bash
cd ../mcp-http-server
go run . --addr :8080
```

2. Configurar el cliente en `.kairos/settings.json` (ya incluido).

## Ejecutar

```bash
cd examples/07-multi-agent-mcp

# Con mock provider (recomendado para testing)
go run .

# Con Ollama (requiere ollama serve)
USE_OLLAMA=1 go run .

# Con timeout ajustado para modelos lentos
TIMEOUT_SECONDS=180 USE_OLLAMA=1 go run .
```

## Qué hace el ejemplo

1. **Descubre tools** del servidor MCP remoto
2. **Llama directamente** al tool `echo` via cliente MCP
3. **Ejecuta el agente** que puede usar las tools descubiertas

## Arquitectura

```
┌─────────────────┐     MCP/HTTP     ┌─────────────────┐
│  Agent Client   │ ◄──────────────► │  MCP Server     │
│  (este ejemplo) │                  │  (mcp-http-srv) │
└─────────────────┘                  └─────────────────┘
        │                                    │
        │                                    │
        ▼                                    ▼
   Descubre tools                     Expone tools:
   Llama remotamente                  - echo
                                      - fetch
                                      - filesystem
```

## Código clave

```go
// Crear cliente MCP HTTP
client, _ := mcp.NewClientWithStreamableHTTPProtocol(
    "http://localhost:8080/mcp",
    "2024-11-05",
)
defer client.Close()

// Descubrir herramientas
tools, _ := client.ListTools(ctx)
for _, tool := range tools {
    fmt.Printf("Tool: %s\n", tool.Name)
}

// Llamar a una herramienta
result, _ := client.CallTool(ctx, "echo", map[string]any{
    "message": "Hello from Kairos!",
})
```

## Troubleshooting

### Error: "connection refused"

El servidor MCP no está corriendo. Ejecutar primero:
```bash
cd ../mcp-http-server && go run . --addr :8080
```

### Error: "TIMEOUT" con Ollama

El modelo no está respondiendo con el formato esperado. Usar mock:
```bash
USE_OLLAMA=0 go run .
```

## Siguiente paso

→ [08-governance-policies](../08-governance-policies/) para control de acceso
