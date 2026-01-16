# 04 - Skills Agent

Agente con skills siguiendo la especificación [AgentSkills](https://agentskills.io/specification).

## Qué aprenderás

- La especificación AgentSkills y su formato SKILL.md
- Cargar skills desde un directorio
- Cómo el agente usa los skills para enriquecer sus capacidades
- Progressive disclosure: metadata → instructions → resources

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
allowed-tools: Bash(pdf:*) Read
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
| `allowed-tools` | No | Tools pre-aprobados (experimental) |

### Progressive disclosure

1. **Metadata** (~100 tokens): `name` y `description` se cargan al inicio
2. **Instructions** (<5000 tokens): El body de SKILL.md se carga al activar
3. **Resources**: Archivos en `scripts/`, `references/`, `assets/` se cargan bajo demanda

## Código clave

```go
// Cargar skills desde directorio
a, err := agent.New("skills-agent", llmProvider,
    agent.WithSkillsFromDir("./skills"),
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

## Siguiente paso

→ [05-mcp-agent](../05-mcp-agent/) para tools via Model Context Protocol
