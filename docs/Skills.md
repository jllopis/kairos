# Skills (AgentSkills)

Kairos implementa el estándar [AgentSkills](https://agentskills.io/specification) para
proporcionar capacidades semánticas a los agentes IA.

## Concepto clave

Los Skills son **tools para LLMs** que implementan **progressive disclosure**:

1. **Metadata** (~100 tokens): El LLM ve `name` y `description` como tool definition
2. **Instructions** (Body): Se inyectan cuando el LLM activa el skill
3. **Resources**: Scripts, referencias y assets se cargan bajo demanda

## Estructura

```
skills/
  pdf-processing/
    SKILL.md
    scripts/          # Opcional
    references/       # Opcional
    assets/           # Opcional
```

El nombre del skill debe coincidir con el nombre del directorio.

## Ejemplo de SKILL.md

```md
---
name: pdf-processing
description: Extrae texto y tablas de PDFs y rellena formularios.
license: Apache-2.0
compatibility: Requiere pdftotext
metadata:
  author: example-org
---

Usa este skill cuando el usuario mencione PDFs o formularios.

## Pasos
1. Identificar el archivo PDF
2. Extraer texto con pdftotext
3. Formatear resultado
```

## Carga en el agente

```go
agent.New("demo-agent", llmProvider,
  agent.WithSkillsFromDir("./skills"),
)
```

## Cómo funciona

Cuando se cargan skills con `WithSkillsFromDir`:

1. Se crean `SkillTool` que implementan `core.Tool`
2. Cada skill se expone al LLM como un tool callable
3. El LLM puede invocar el skill con diferentes acciones:

```json
// Activar skill (obtener instrucciones)
{"action": "activate"}

// Listar recursos disponibles
{"action": "list_resources"}

// Cargar un recurso específico
{"action": "load_resource", "resource": "scripts/extract.py"}
```

## Filtrado de tools (Governance)

El campo `allowed-tools` del frontmatter está disponible pero el filtrado de tools
debe hacerse a través del módulo de governance, no en los skills:

```go
// Configurar filtrado via governance
toolFilter := governance.NewToolFilter(
    governance.WithAllowlist([]string{"pdf-processing", "Bash(pdf:*)"}),
)

agent.New("demo-agent", llmProvider,
    agent.WithSkillsFromDir("./skills"),
    agent.WithToolFilter(toolFilter),
)
```

Esto separa:
- **Skills**: Definición de capacidades semánticas
- **Governance**: Control de acceso a tools

## Ejemplo completo

Ver `examples/04-skills-agent` para un ejemplo runnable con un directorio de skills.
