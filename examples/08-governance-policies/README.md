# 08 - Governance & Policies

Control de acceso y políticas para restringir qué puede hacer un agente.

## Qué aprenderás

- Definir políticas Allow/Deny por tipo y nombre
- Aplicar governance a llamadas de herramientas MCP
- Bloquear acciones antes de ejecutarlas
- Separar "qué puede hacer" de "qué sabe hacer"

## Requisitos

Levantar el servidor MCP HTTP:
```bash
cd ../mcp-http-server
go run . --addr :8080
```

## Ejecutar

```bash
cd examples/08-governance-policies
go run .
```

Salida esperada:
```
attempting tool call (should be denied): echo
policy denied tool call: blocked by policy
```

## Configuración de políticas

Las políticas se definen en `.kairos/settings.json`:

```json
{
  "governance": {
    "policies": [
      {
        "id": "deny-all-tools",
        "effect": "deny",
        "type": "tool",
        "name": "*",
        "reason": "blocked by policy"
      }
    ]
  }
}
```

### Campos de una política

| Campo | Descripción | Valores |
|-------|-------------|---------|
| `id` | Identificador único | string |
| `effect` | Acción a tomar | `allow`, `deny`, `pending` |
| `type` | Tipo de recurso | `tool`, `action` |
| `name` | Nombre o patrón | `*` para todos, o nombre exacto |
| `reason` | Mensaje de error | string |

## Código clave

```go
// Cargar configuración con políticas
cfg, _ := config.LoadWithCLI(os.Args[1:])

// Crear engine de políticas desde config
policy := governance.RuleSetFromConfig(cfg.Governance)

// Cliente MCP con policy engine
client, _ := mcp.NewClientWithStreamableHTTPProtocol(url, version,
    mcp.WithPolicyEngine(policy),
)

// La llamada será bloqueada por la política
_, err := client.CallTool(ctx, "echo", args)
// err: "blocked by policy"
```

## Ejemplos de políticas

### Denegar todas las herramientas
```json
{"effect": "deny", "type": "tool", "name": "*", "reason": "tools disabled"}
```

### Permitir solo lectura
```json
[
  {"effect": "allow", "type": "tool", "name": "read_file"},
  {"effect": "allow", "type": "tool", "name": "list_directory"},
  {"effect": "deny", "type": "tool", "name": "*", "reason": "only read operations allowed"}
]
```

### Denegar escritura
```json
{"effect": "deny", "type": "tool", "name": "write_file", "reason": "write disabled"}
```

## Flujo de evaluación

```
1. Cliente MCP intenta llamar tool X
2. PolicyEngine.Evaluate("tool", "X")
3. Busca primera regla que matchee tipo y nombre
4. Si effect=allow → ejecuta tool
5. Si effect=deny → retorna error con reason
6. Si no match → permite por defecto
```

## Siguiente paso

→ [09-error-handling](../09-error-handling/) para manejo de errores tipado
