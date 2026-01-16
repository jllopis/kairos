# Ejemplo 13: MCP Connection Pool

## Qué aprenderás

- Crear un pool de conexiones MCP compartido
- Registrar servidores MCP (stdio y HTTP)
- Compartir conexiones entre múltiples agentes
- Gestión eficiente de recursos con reference counting
- Monitorear estadísticas del pool

## Cuándo usar el pool

| Escenario | Recomendación |
|-----------|---------------|
| Un solo agente | Conexión directa (`mcp.NewClientWithStdio`) |
| Múltiples agentes, mismo servidor | **Pool compartido** |
| Microservicios con agentes | **Pool compartido** |
| Tests unitarios | Conexión directa o mock |

## Arquitectura

```
┌─────────────────────────────────────────┐
│           Kairos Runtime                │
│  ┌─────────────────────────────────┐    │
│  │     MCP Connection Pool         │    │
│  │  ┌─────────┐  ┌─────────┐       │    │
│  │  │filesystem│ │ github  │  ...  │    │
│  │  └─────────┘  └─────────┘       │    │
│  └─────────────────────────────────┘    │
│                  ▲                       │
│     ┌────────────┼────────────┐         │
│     │            │            │         │
│  ┌──┴───┐   ┌────┴───┐   ┌───┴───┐     │
│  │Agent1│   │Agent2  │   │Agent3 │     │
│  └──────┘   └────────┘   └───────┘     │
└─────────────────────────────────────────┘
```

## Ejecutar

```bash
go run main.go
```

**Nota**: Este ejemplo registra servidores MCP pero no los ejecuta realmente.
En un entorno real, necesitarías tener los servidores MCP corriendo.

## Código clave

### Crear el pool

```go
mcpPool := pool.New(
    pool.WithMaxConnectionsPerServer(5),
    pool.WithHealthCheckInterval(30 * time.Second),
    pool.WithIdleTimeout(5 * time.Minute),
)
defer mcpPool.Close()
```

### Registrar servidores

```go
// Servidor stdio (Kairos gestiona el proceso)
mcpPool.RegisterStdio("filesystem", "npx", []string{"-y", "@mcp/server-filesystem"})

// Servidor HTTP (ya debe estar corriendo)
mcpPool.RegisterHTTP("github", "http://localhost:8080/mcp")
```

### Usar desde un agente

```go
// Obtener conexión (reutiliza si existe)
client, err := mcpPool.Get(ctx, "filesystem")
if err != nil {
    // handle error
}

// Usar el cliente
tools, _ := client.ListTools(ctx)

// Liberar (no cierra, solo decrementa ref count)
mcpPool.Release("filesystem", client)
```

## Métricas disponibles

```go
stats := mcpPool.Stats()
// stats.RegisteredServers  - Servidores configurados
// stats.ActiveConnections  - Conexiones en uso
// stats.TotalConnections   - Total de conexiones creadas
// stats.ConnectionErrors   - Errores al conectar
// stats.HealthChecksPassed - Health checks exitosos
// stats.HealthChecksFailed - Health checks fallidos
```

## Siguiente paso

Ver `examples/12-production-layout/` para un ejemplo completo de estructura de proyecto enterprise con pool MCP integrado.
