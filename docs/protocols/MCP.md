# Model Context Protocol (MCP)

MCP define un estándar abierto para describir y ejecutar herramientas que un
agente puede usar. En Kairos, MCP es el mecanismo principal para integrar
capacidades externas sin APIs propietarias.

## En Kairos

Kairos ofrece cliente y servidor MCP de primera clase. Las skills se resuelven
a herramientas MCP disponibles y un agente puede exponer sus propias herramientas
vía MCP.

---

## Transportes MCP

MCP define dos transportes principales. **Es importante entender la diferencia** para configurar correctamente tus servidores.

### Transporte `stdio` (Kairos gestiona el proceso)

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

**Comportamiento:**

- Kairos **inicia el proceso** MCP al crear el agente
- Kairos **termina el proceso** al cerrar el agente (`agent.Close()`)
- La comunicación es via stdin/stdout del proceso

**Cuándo usarlo:**

- Servidores MCP empaquetados como CLI (npm, binarios)
- Desarrollo local
- Cuando cada agente necesita su propia instancia

---

### Transporte `http` (Servidor externo)

```json
{
  "mcp": {
    "servers": {
      "my-api": {
        "transport": "http",
        "url": "http://localhost:8080/mcp"
      }
    }
  }
}
```

**Comportamiento:**

- El servidor MCP **debe estar ejecutándose** antes de crear el agente
- Kairos **solo se conecta**, no gestiona el lifecycle
- Si el servidor no está disponible, la creación del agente falla

**Cuándo usarlo:**

- Servidores MCP compartidos entre múltiples agentes
- Servidores desplegados como servicios (Docker, Kubernetes)
- Integración con APIs empresariales

---

### Comparativa de Transportes

| Aspecto | `stdio` | `http` |
|---------|---------|--------|
| Lifecycle | Kairos lo gestiona | Externo |
| Inicio | Automático al crear agente | Manual (debe estar corriendo) |
| Cierre | Automático en `Close()` | No aplica |
| Escalabilidad | 1 proceso por agente | Compartible entre agentes |
| Desarrollo | Ideal | Requiere setup previo |
| Producción | Posible, pero más procesos | Recomendado |

---

## Configuración

Kairos busca configuración MCP en estos paths (en orden):

1. `./.kairos/settings.json` (directorio actual)
2. `$HOME/.kairos/settings.json`
3. `$XDG_CONFIG_HOME/kairos/settings.json`

### Estructura del archivo

```json
{
  "mcp": {
    "servers": {
      "<nombre-servidor>": {
        "transport": "stdio" | "http",
        "command": "<comando>",           // Solo stdio
        "args": ["<arg1>", "<arg2>"],     // Solo stdio
        "url": "http://..."               // Solo http
      }
    }
  }
}
```

---

## Ejemplo: Múltiples Servidores

```json
{
  "mcp": {
    "servers": {
      "filesystem": {
        "transport": "stdio",
        "command": "npx",
        "args": ["-y", "@modelcontextprotocol/server-filesystem", "/home/user/docs"]
      },
      "github": {
        "transport": "stdio",
        "command": "npx",
        "args": ["-y", "@modelcontextprotocol/server-github"]
      },
      "company-api": {
        "transport": "http",
        "url": "https://mcp.internal.company.com/v1"
      }
    }
  }
}
```

---

## Uso en Código

```go
import (
    "github.com/jllopis/kairos/pkg/agent"
    "github.com/jllopis/kairos/pkg/config"
)

// Cargar configuración
cfg, _ := config.LoadWithCLI(os.Args[1:])

// Crear agente con servidores MCP
ag, err := agent.New("my-agent", provider,
    agent.WithMCPServerConfigs(cfg.MCP.Servers),
)
if err != nil {
    // Error si algún servidor HTTP no está disponible
    log.Fatal(err)
}
defer ag.Close() // Importante: cierra conexiones stdio

// Listar herramientas disponibles
tools, _ := ag.MCPTools(ctx)
for _, t := range tools {
    fmt.Printf("Tool: %s - %s\n", t.Name, t.Description)
}

// El agente usa las herramientas automáticamente
result, _ := ag.Run(ctx, "Busca archivos .go en el proyecto")
```

---

## Servidores MCP Populares

| Servidor | Comando | Descripción |
|----------|---------|-------------|
| Filesystem | `npx -y @modelcontextprotocol/server-filesystem <path>` | Lectura/escritura de archivos |
| GitHub | `npx -y @modelcontextprotocol/server-github` | API de GitHub |
| PostgreSQL | `npx -y @modelcontextprotocol/server-postgres` | Consultas SQL |
| Fetch | `docker run -i --rm mcp/fetch` | HTTP requests |
| Brave Search | `npx -y @anthropics/mcp-server-brave-search` | Búsqueda web |

Ver más en [MCP Servers Directory](https://github.com/modelcontextprotocol/servers).

---

## Troubleshooting

### Error: "connection refused" con transporte HTTP

```
failed to create agent: mcp server "my-api": transport error: connection refused
```

**Causa**: El servidor MCP no está ejecutándose.

**Solución**: Iniciar el servidor antes de ejecutar el agente, o usar transporte `stdio` si el servidor lo soporta.

### Error: "command not found" con transporte stdio

```
failed to create agent: mcp server "filesystem": exec: "npx": executable file not found
```

**Causa**: El comando especificado no está en el PATH.

**Solución**: Instalar la dependencia (`npm install -g npx`) o usar path absoluto.

---

## Relación con A2A

A2A conecta agentes entre sí. MCP conecta agentes con herramientas.
Son complementarios: un agente puede delegar vía A2A y ejecutar tools vía MCP.

---

## Especificación Oficial

https://modelcontextprotocol.io/
