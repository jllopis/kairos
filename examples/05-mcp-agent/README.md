# 05 - MCP Agent

Agente que descubre y usa herramientas via Model Context Protocol (MCP).

## Qué aprenderás

- Configurar servidores MCP desde archivo de configuración
- Descubrir herramientas dinámicamente
- Ejecutar tools MCP desde el agente
- Transportes stdio y HTTP

## Requisitos

- Node.js (para `npx`)
- Ollama ejecutándose (o usar `USE_OLLAMA=0` para mock)

## Configuración

El ejemplo incluye `.kairos/settings.json` con un servidor MCP preconfigurado:

```json
{
  "mcp": {
    "servers": {
      "filesystem": {
        "transport": "stdio",
        "command": "npx",
        "args": ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"]
      }
    }
  }
}
```

Kairos busca configuración en estos paths (en orden):

1. `./.kairos/settings.json` (directorio actual)
2. `$HOME/.kairos/settings.json`
3. `$XDG_CONFIG_HOME/kairos/settings.json`

## Ejecutar

```bash
cd examples/05-mcp-agent

# Con Ollama (requiere ollama serve)
go run .

# Con mock provider (sin LLM real)
USE_OLLAMA=0 go run .
```

## Transportes MCP

| Transporte | Cuándo usarlo | Ejemplo |
|------------|---------------|---------|
| `stdio` | Proceso local que Kairos inicia | filesystem, github |
| `http` | Servidor ya ejecutándose | APIs remotas |

### Ejemplo stdio (Kairos inicia el proceso)

```json
{
  "filesystem": {
    "transport": "stdio",
    "command": "npx",
    "args": ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"]
  }
}
```

### Ejemplo HTTP (servidor externo)

```json
{
  "my-api": {
    "transport": "http",
    "url": "http://localhost:8080/mcp"
  }
}
```

> **Nota**: Para HTTP, el servidor debe estar ejecutándose antes de iniciar el agente.

## Código clave

```go
// Cargar configuración (busca settings.json automáticamente)
cfg, _ := config.LoadWithCLI(os.Args[1:])

// Agente con servidores MCP configurados
ag, _ := agent.New("mcp-agent", provider,
    agent.WithMCPServerConfigs(cfg.MCP.Servers),
)
defer ag.Close() // Importante: cierra conexiones MCP

// Listar tools disponibles
tools, _ := ag.MCPTools(ctx)
for _, t := range tools {
    fmt.Printf("Tool: %s\n", t.Name)
}

// El agente usa las tools MCP automáticamente
result, _ := ag.Run(ctx, "Lista archivos en /tmp")
```

## Servidores MCP populares

| Servidor | Comando |
|----------|---------|
| Filesystem | `npx -y @modelcontextprotocol/server-filesystem <path>` |
| GitHub | `npx -y @modelcontextprotocol/server-github` |
| PostgreSQL | `npx -y @modelcontextprotocol/server-postgres` |
| Fetch | `docker run -i --rm mcp/fetch` |

## Siguiente paso

→ [06-explicit-planner](../06-explicit-planner/) para orquestación con grafos
