# Plan de Mejora de Developer Experience (DX) - Kairos

## Resumen Ejecutivo

Este documento analiza las propuestas de mejora de DX recibidas y las adapta al contexto real de Kairos, manteniendo su filosofía "architecture-first" y su posicionamiento como **plataforma para construir sistemas de agentes**, no como un framework opinionado.

### Principio rector

> **Kairos no debe ocultar la arquitectura, pero sí debe eliminar la fricción innecesaria.**

---

## Análisis del Estado Actual

### Fortalezas de Kairos (a preservar)

1. **Arquitectura explícita**: Componentes bien definidos (Agent, LLM Provider, Memory, Planner, Governance)
2. **Flexibilidad**: Todo es configurable mediante Options pattern
3. **Observabilidad nativa**: OpenTelemetry integrado desde el core
4. **Interoperabilidad**: Soporte A2A y MCP estándar
5. **CLI funcional**: Comandos operativos (status, tasks, traces, approvals, mcp, registry)

### Gaps identificados

1. **Onboarding**: No hay `kairos init` para scaffolding
2. **Ejemplos**: Existen pero no están numerados/organizados progresivamente
3. **Documentación DX**: Falta guía clara de "primeros 5 minutos"
4. **Tooling de desarrollo**: No hay `kairos validate`, `kairos explain`, `kairos graph`

---

## Decisión Arquitectónica: ¿Capa adicional?

### Análisis

Se propuso considerar una **librería adicional** (capa superior más opinionada). Tras analizar Kairos:

**NO se recomienda crear una capa adicional porque:**

1. **El Options pattern ya proporciona abstracción gradual**:
   ```go
   // Mínimo
   agent.New("mi-agente", llmProvider)
   
   // Con más features
   agent.New("mi-agente", llmProvider,
       agent.WithMemory(mem),
       agent.WithPolicyEngine(gov),
       agent.WithTools(tools),
   )
   ```

2. **El problema no es la complejidad del API**, sino la fricción de setup inicial

3. **Una capa adicional violaría el principio "explicit over implicit"**

### Recomendación

En lugar de una capa adicional, mejorar DX mediante:
- **Scaffolding inteligente** (templates que enseñan)
- **CLI enriquecido** (comandos de introspección)
- **Ejemplos canónicos** (aprender leyendo código)

---

## Roadmap DX Propuesto

### Fase 0: Fundamentos (1-2 semanas)

**Objetivo**: Reducir fricción inicial sin añadir código.

| Tarea | Descripción | Impacto |
|-------|-------------|---------|
| README renovado | Quick start honesto + diagrama de arquitectura | Alto |
| "Qué es / Qué no es" | Sección clara de posicionamiento | Medio |
| Organizar `/examples` | Numeración progresiva (01-hello, 02-memory, etc.) | Alto |

**Estructura propuesta de examples:**
```
examples/
├── 01-hello-agent/                                       # Mínimo viable
├── 02-agent-with-memory/                                 # + memoria semántica
├── 03-agent-with-tools/                                  # + tools locales, SKILLs y AGENTS.md
├── 04-agent-with-mcp/                                    # + tools MCP
├── 05-agent-with-policies/                               # + governance
├── 06-agent-with-planner/                                # + planner explícito
├── 07-multi-agent-a2a/                                   # + comunicación entre agentes
├── 08-multi-agent-a2a-with-telemetry-observability/      # + comunicación entre agentes
└── 09-production-layout/                                 # Estructura enterprise completa
```

Cada ejemplo: <200 líneas, README con "qué aprender aquí", sin helpers ocultos.

---

### Fase 1: CLI Scaffolding (2-3 semanas)

**Objetivo**: `kairos init` que genera proyectos aprendibles.

#### Comando: `kairos init`

```bash
kairos init <dir> --module <go-module> [--type <archetype>] [--llm <provider>]
```

**Flags:**
- `--module` (required): Go module path
- `--type`: `assistant` | `tool-agent` | `coordinator` | `policy-heavy` (default: assistant)
- `--llm`: `ollama` | `mock` (default: ollama)
- `--mcp`: Incluir wiring MCP
- `--a2a`: Incluir endpoint A2A

**Estructura generada (arquetipo `assistant`):**
```
my-agent/
├── cmd/agent/main.go           # Entrypoint limpio
├── internal/
│   ├── app/app.go              # Wiring explícito de componentes
│   ├── config/config.go        # Loader de configuración
│   └── observability/otel.go   # Setup OTEL
├── config/config.yaml          # Config runtime
├── Makefile                    # run, build, test, tidy
├── go.mod
└── README.md                   # Quick start del proyecto
```

**Principio clave**: El scaffold **enseña Kairos**. Todo el código generado es legible, comentado y editable.

#### Arquetipos

| Tipo | Características |
|------|-----------------|
| `assistant` | Memoria conversacional + LLM básico |
| `tool-agent` | + tools locales/MCP preconfigurados |
| `coordinator` | + planner + A2A para delegación |
| `policy-heavy` | + governance explícito con políticas ejemplo |

---

### Fase 2: CLI Operativo (2-3 semanas)

**Objetivo**: Comandos para desarrollo y debugging.

#### `kairos run`

```bash
kairos run [--config config.yaml] [--profile dev|prod]
```

Ejecuta un agente con hot-reload de config (útil para desarrollo).

#### `kairos validate`

```bash
kairos validate [--config config.yaml]
```

Validación estática:
- ✅ Config YAML válida
- ✅ Policies bien formadas
- ✅ LLM provider alcanzable
- ✅ MCP servers configurados correctamente
- ✅ SKILLs disponibles correctos

---

### Fase 3: CLI de Introspección ✅ COMPLETADA

**Objetivo**: Herramientas para entender qué hace el agente.

**Estado**: Implementado en `cmd/kairos/explain.go`, `cmd/kairos/graph.go`, `cmd/kairos/adapters.go`.

#### `kairos explain`

```bash
kairos explain [--agent <id>] [--skills <dir>]
```

Output:
```
Agent: my-assistant
├── LLM: ollama (llama3.1)
├── Memory: inmemory
├── Governance: enabled
│   ├── Policy: be-brief
│   └── Policy: no-secrets
├── Tools: 3
│   ├── get_weather (MCP: weather-server)
│   ├── search_docs (MCP: filesystem)
│   └── send_email (MCP: email-server)
├── Skills: 2
│   ├── pdf-processing: Extract text from PDFs
│   └── summarization: Summarize documents
└── A2A: disabled
```

#### `kairos graph`

```bash
kairos graph --path workflow.yaml [--output mermaid|dot|json]
```

Genera visualizaciones del planner DAG en Mermaid, Graphviz DOT, o JSON.

#### `kairos adapters`

```bash
kairos adapters list [--type llm|memory|mcp|a2a|telemetry]
kairos adapters info <name>
```

Catálogo de providers disponibles con detalles de configuración.

---

### Fase 4: DX Enterprise (4-6 semanas)

**Objetivo**: Facilitar adopción en equipos grandes.

#### MCP Runtime Compartido

Actualmente cada agente gestiona sus propias conexiones MCP. En escenarios multi-agente o enterprise, esto genera:

- **Duplicación de procesos**: N agentes → N instancias del mismo MCP server
- **Ineficiencia de recursos**: Cada agente inicia/cierra conexiones
- **Complejidad operativa**: Difícil gestionar lifecycle de MCPs distribuidos

**Propuesta**: Runtime de Kairos que gestione MCPs compartidos entre agentes.

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

**Beneficios:**
- Un proceso MCP compartido por múltiples agentes
- Lifecycle gestionado centralmente (start/stop/health)
- Posibilidad de pooling de conexiones HTTP
- Métricas y observabilidad unificadas

**Consideraciones de diseño:**
- Backward compatible: agentes individuales siguen funcionando igual
- Opt-in: el runtime compartido es opcional
- El comportamiento por defecto no cambia

**Impacto**: Estratégico para escenarios enterprise y multi-agente.

#### Config layering

```
config/
├── config.yaml           # Base
├── config.dev.yaml       # Override desarrollo
└── config.prod.yaml      # Override producción
```

Con merge explícito y documentado.

#### Templates corporativos

Repo template con:
- Observabilidad preconfigurada (Grafana Cloud/Grafana Alloy/Datadog/New Relic)
- Policies de compliance
- CI/CD (GitHub Actions/Bitbucket Pipelines/AWS Code Pipelines)
- Dockerfile optimizado

---

## Implementación Técnica

### Integración en CLI actual

El CLI actual usa `flag` + `switch cmd`. Para mantener consistencia:

```go
// cmd/kairos/main.go
switch cmd {
case "init":
    runInit(global, args[1:])
case "run":
    runRun(global, args[1:])
case "validate":
    runValidate(global, args[1:])
case "explain":
    runExplain(global, args[1:])
// ... casos existentes
}
```

### Estructura de archivos nuevos

```
cmd/kairos/
├── main.go              # Existente, añadir cases
├── init.go              # Nuevo: kairos init
├── run.go               # Nuevo: kairos run  
├── validate.go          # Nuevo: kairos validate
├── explain.go           # Nuevo: kairos explain
└── scaffold/
    ├── scaffold.go      # Lógica de generación
    └── templates/       # embed.FS con .tmpl
        ├── assistant/
        ├── tool-agent/
        ├── coordinator/
        └── policy-heavy/
```

### Templates con `embed.FS`

```go
//go:embed templates/*
var templatesFS embed.FS

func generateProject(dir string, opts Options) error {
    // Usar text/template para render
}
```

---

## Lo que NO hacer

❌ **No añadir:**
- DSLs mágicos
- JSON "todo en uno"
- Configs opacas que reemplacen Go
- Abstracciones que oculten la arquitectura

❌ **No competir con:**
- LangChain (simplicidad sobre control)
- BeeAI (batteries included)

**Kairos gana siendo explícito**, no "más fácil".

---

## Priorización

| Fase | Esfuerzo | Impacto | Prioridad |
|------|----------|---------|-----------|
| 0 - Fundamentos | Bajo | Alto | 🔴 Inmediata |
| 1 - Scaffolding | Medio | Muy alto | 🔴 Inmediata |
| 2 - CLI Operativo | Medio | Alto | 🟡 Corto plazo |
| 3 - Introspección | Alto | Muy alto | 🟡 Corto plazo |
| 4 - Enterprise | Alto | Estratégico | 🟢 Medio plazo |

---

## Métricas de Éxito

1. **Time to first agent**: <5 minutos con `kairos init`
2. **Comprensión arquitectónica**: Usuario entiende componentes tras leer 01-hello-agent
3. **Debugging**: `kairos explain` reduce tiempo de troubleshooting 50%
4. **Adopción enterprise**: Template corporativo funcional en <1 hora

---

## Conclusión

El DX de Kairos puede mejorar significativamente sin sacrificar su filosofía:

1. **Scaffolding que enseña** (no que oculta)
2. **CLI que introspecciona** (no que abstrae)
3. **Ejemplos que progresan** (no que abruman)

> **Kairos no debe esconder la complejidad. Debe domesticarla.**

---

**Autor**: Plan generado tras análisis de feedback de usuarios  
**Fecha**: 2026-01-15  
**Estado**: Propuesta para revisión
