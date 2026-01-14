# Model Context Protocol (MCP)

MCP define un estándar abierto para describir y ejecutar herramientas que un
agente puede usar. En Kairos, MCP es el mecanismo principal para integrar
capacidades externas sin APIs propietarias.

## En Kairos

Kairos ofrece cliente y servidor MCP de primera clase. Las skills se resuelven
a herramientas MCP disponibles y un agente puede exponer sus propias herramientas
vía MCP.

## Patrón de integración

El agente consulta tools disponibles en los MCP conectados, filtra herramientas
por skills declaradas y ejecuta tools antes de reintegrar resultados en su loop.

## Ejemplo mínimo

Ejemplo de cliente MCP vía stdio:

```go
client, err := mcp.NewClientWithStdioProtocol("node", []string{serverPath}, "2024-11-05")
if err != nil {
  // handle error
}
defer client.Close()

agent, err := agent.New("demo-agent", llmProvider,
  agent.WithMCPClients(client),
)
```

Config de ejemplo:

```json
{
  "mcp": {
    "servers": {
      "fetch": {
        "transport": "stdio",
        "command": "docker",
        "args": ["run", "-i", "--rm", "mcp/fetch"]
      }
    }
  }
}
```

## Relación con A2A

A2A conecta agentes entre sí. MCP conecta agentes con herramientas.
Son complementarios: un agente puede delegar vía A2A y ejecutar tools vía MCP.

## Siguiente paso

Mientras no haya una implementación completa, recomendamos revisar la
especificación oficial:
https://modelcontextprotocol.io/
