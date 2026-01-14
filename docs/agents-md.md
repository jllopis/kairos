# AGENTS.md

AGENTS.md es el archivo donde dejas claro cómo deben comportarse los agentes en
un repo. Kairos lo carga al arrancar, sin configuración extra.

## Para que sirve

Te ayuda a dejar por escrito:

- Reglas de seguridad y calidad.
- Convenciones del repo y del stack.
- Límites de lo que un agente puede o no puede hacer.

## Cómo lo usa Kairos

El runtime busca `AGENTS.md`, lo incorpora al contexto base y aplica esas reglas
durante la ejecución.

## Ejemplo de AGENTS.md

```md
# AGENTS.md

## Reglas
- No tocar secretos ni credenciales.
- No ejecutar comandos destructivos sin confirmación.
- Mantener cambios pequeños y explicados.

## Stack
- Go, Markdown
- Comandos: go test ./..., go run ./examples/hello-agent
```

## Uso en el agente

No necesitas código adicional: el contenido se añade al prompt del sistema al
crear el agente. Si quieres forzar instrucciones concretas, puedes inyectarlas
con `agent.WithAGENTSInstructions(...)`.

Ejemplo mínimo:

```go
doc := &governance.AgentInstructions{
  Path: "AGENTS.md",
  Raw:  "# AGENTS.md\\n\\n- No ejecutar comandos destructivos.",
}

agent.New("demo-agent", llmProvider,
  agent.WithAGENTSInstructions(doc),
)
```

## Consejos rápidos

- Mantenlo corto y accionable.
- Separa reglas obligatorias de sugerencias.
- Revísalo cuando cambie la arquitectura.

## Relación con governance

AGENTS.md define el marco de comportamiento. Las políticas de governance aplican
esas reglas en tiempo de ejecución. Para ver ejemplos y configuración, consulta
la [Guía de governance](governance-usage.md).
