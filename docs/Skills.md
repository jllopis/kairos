# Skills (AgentSkills)

Kairos soporta el estándar AgentSkills. Cada skill vive en una carpeta con un
`SKILL.md` y se carga desde el agente.

## Estructura

```
skills/
  pdf-processing/
    SKILL.md
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
allowed-tools: Bash(pdf:*) Bash(ocr:*)
---

Usa este skill cuando el usuario mencione PDFs o formularios.
```

## Carga en el agente

```go
agent.New("demo-agent", llmProvider,
  agent.WithSkillsFromDir("./skills"),
)
```

Si `allowed-tools` está presente, se usa como allowlist de tools.

## Ejemplo completo

Ver `examples/skills-agent` para un ejemplo runnable con un directorio de skills.
