# 04 - Skills Agent

Agente con skills siguiendo la especificación [AgentSkills](https://agentskills.io/specification).

## Qué aprenderás

- La especificación AgentSkills y su formato SKILL.md
- Cargar skills desde un directorio como tools para el LLM
- Progressive disclosure: metadata → instructions → resources
- Filtrado de tools mediante governance (no en skills)

## Ejecutar

```bash
cd examples/04-skills-agent
go run .
```

## Especificación AgentSkills

Un skill es un directorio con un archivo `SKILL.md` obligatorio:

```
skills/
└── pdf-processing/
    └── SKILL.md          # Requerido
```

### Formato SKILL.md

El archivo debe contener frontmatter YAML seguido de contenido Markdown:

```yaml
---
name: pdf-processing
description: Extract text and tables from PDF files, fill forms, merge documents.
license: Apache-2.0
compatibility: Requires pdftotext
metadata:
  author: example-org
  version: "1.0"
---

# Instrucciones

Use este skill cuando el usuario mencione PDFs, formularios o extracción de documentos.

## Pasos
1. Identificar el archivo PDF
2. Extraer texto con pdftotext
3. Formatear resultado
```

### Campos del frontmatter

| Campo | Requerido | Descripción |
|-------|-----------|-------------|
| `name` | Sí | 1-64 chars, lowercase, puede contener `-` |
| `description` | Sí | 1-1024 chars, describe qué hace y cuándo usarlo |
| `license` | No | Licencia del skill |
| `compatibility` | No | Requisitos de entorno |
| `metadata` | No | Metadatos adicionales (key-value) |

> **Nota**: El campo `allowed-tools` de la especificación AgentSkills está disponible pero
> el filtrado de tools debe hacerse a través del módulo de governance, no en los skills.

### Progressive disclosure

Los skills implementan carga progresiva según la especificación AgentSkills:

1. **Metadata** (~100 tokens): `name` y `description` se exponen al LLM como tool definition
2. **Instructions** (<5000 tokens): El body de SKILL.md se inyecta cuando el LLM activa el skill
3. **Resources**: Archivos en `scripts/`, `references/`, `assets/` se cargan bajo demanda

## Skills como Tools

Los skills se exponen al LLM como tools nativos. Cuando el LLM decide usar un skill:

```json
{
  "action": "activate"
}
```

Recibe las instrucciones completas del skill (el Body del SKILL.md).

También puede:
- `{"action": "list_resources"}` - Ver recursos disponibles
- `{"action": "load_resource", "resource": "scripts/extract.py"}` - Cargar un recurso específico

## Código clave

```go
// Cargar skills desde directorio (se exponen como tools al LLM)
a, err := agent.New("skills-agent", llmProvider,
    agent.WithSkillsFromDir("./skills"),
)

// Opcional: Configurar filtrado de tools via governance
toolFilter := governance.NewToolFilter(
    governance.WithAllowlist([]string{"pdf-processing", "Bash(pdf:*)"}),
)
a, err := agent.New("skills-agent", llmProvider,
    agent.WithSkillsFromDir("./skills"),
    agent.WithToolFilter(toolFilter),
)

// Ver skills cargados
for _, skill := range a.Skills() {
    fmt.Printf("Skill: %s - %s\n", skill.Name, skill.Description)
}
```

## Estructura opcional de un skill

```
pdf-processing/
├── SKILL.md              # Requerido: frontmatter + instrucciones
├── scripts/              # Opcional: código ejecutable
│   └── extract.py
├── references/           # Opcional: documentación adicional
│   └── REFERENCE.md
└── assets/               # Opcional: recursos estáticos
    └── template.pdf
```

## Filtrado de tools (Governance)

El filtrado de tools se gestiona a través del módulo `governance.ToolFilter`:

```go
// Crear filtro con allowlist
filter := governance.NewToolFilter(
    governance.WithAllowlist([]string{"tool-a", "tool-b"}),
)

// O con denylist
filter := governance.NewToolFilter(
    governance.WithDenylist([]string{"dangerous-tool"}),
)

// O combinado con policy engine
filter := governance.NewToolFilter(
    governance.WithPolicyEngine(policyEngine),
)
```

Esto separa la definición de capacidades (skills) del control de acceso (governance).

## Siguiente paso

→ [05-mcp-agent](../05-mcp-agent/) para tools via Model Context Protocol
