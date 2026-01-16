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

### Fase 0: Fundamentos ✅ COMPLETADA

**Objetivo**: Reducir fricción inicial sin añadir código.

**Estado**: Implementado en commit `b4c008e`.

| Tarea | Descripción | Estado |
|-------|-------------|--------|
| README renovado | Quick start honesto + diagrama de arquitectura | ✅ |
| "Qué es / Qué no es" | Sección clara de posicionamiento | ✅ |
| Organizar `/examples` | Numeración progresiva (01-hello, 02-memory, etc.) | ✅ |

**Estructura implementada de examples:**
```
examples/
├── 01-hello-agent/           # Mínimo viable
├── 02-basic-agent/           # + configuración básica
├── 03-memory-agent/          # + memoria semántica
├── 04-skills-agent/          # + SKILLs
├── 05-mcp-agent/             # + tools MCP
├── 06-explicit-planner/      # + planner explícito
├── 07-multi-agent-mcp/       # + multi-agente
├── 08-governance-policies/   # + governance
├── 09-error-handling/        # + manejo de errores
├── 10-resilience-patterns/   # + patrones de resiliencia
├── 11-observability/         # + observabilidad
├── 12-production-layout/     # Estructura enterprise
└── 13-mcp-pool/              # Pool de conexiones MCP
```

Cada ejemplo incluye README.md con objetivos de aprendizaje.

---

### Fase 1: CLI Scaffolding ✅ COMPLETADA

**Objetivo**: `kairos init` que genera proyectos aprendibles.

**Estado**: Implementado en `cmd/kairos/init.go` y `cmd/kairos/scaffold/`.

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

### Fase 2: CLI Operativo ✅ COMPLETADA

**Objetivo**: Comandos para desarrollo y debugging.

**Estado**: Implementado en `cmd/kairos/run.go` y `cmd/kairos/validate.go`.

#### `kairos run`

```bash
kairos run [--config config.yaml] [--profile dev|prod]
```

Ejecuta un agente con soporte para config layering.

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

### Fase 4: DX Enterprise ✅ COMPLETADA

**Objetivo**: Facilitar adopción en equipos grandes.

**Estado**: Implementado en múltiples commits:
- MCP Pool: `pkg/mcp/pool/` (commit `7014742`)
- Config Layering: `pkg/config/` (commit `7d22404`)
- Corporate Templates: `cmd/kairos/scaffold/templates_corporate.go` (commit `374caac`)

#### MCP Runtime Compartido ✅

Implementado en `pkg/mcp/pool/pool.go`.

```go
// Crear pool compartido
mcpPool := pool.New(
    pool.WithMaxConnectionsPerServer(5),
    pool.WithHealthCheckInterval(30 * time.Second),
)

// Registrar servidores
mcpPool.RegisterStdio("filesystem", "npx", []string{...})
mcpPool.RegisterHTTP("github", "http://localhost:8080/mcp")

// Usar desde agentes
client, _ := mcpPool.Get(ctx, "filesystem")
defer mcpPool.Release("filesystem", client)
```

Características:
- Reference counting para conexiones
- Health checks automáticos
- Cleanup de conexiones idle
- Métricas del pool

Ver `examples/13-mcp-pool/` y `docs/protocols/MCP.md`.

#### Config Layering ✅

Implementado en `pkg/config/config.go`.

```bash
# Uso desde CLI
kairos run --config config/config.yaml --profile dev
kairos run --config config/config.yaml --env prod
```

```go
// Uso programático
cfg, _ := config.LoadWithProfile("config.yaml", "dev")
```

Orden de precedencia:
1. Defaults del framework
2. config.yaml (base)
3. config.dev.yaml (profile)
4. Variables de entorno (KAIROS_*)
5. CLI overrides (--set)

Ver `docs/CONFIGURATION.md`.

#### Templates Corporativos ✅

Implementado con flag `--corporate` en `kairos init`.

```bash
kairos init my-agent --module github.com/org/agent --corporate
```

Genera:
- `.github/workflows/ci.yaml`: CI/CD pipeline
- `Dockerfile`: Multi-stage optimizado
- `docker-compose.yaml`: Stack de desarrollo
- `deploy/otel-collector-config.yaml`: OTEL config
- `deploy/prometheus.yaml`: Prometheus config
- `.golangci.yaml`: Linters config

Ver `docs/CORPORATE_TEMPLATES.md`.

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

| Fase | Esfuerzo | Impacto | Estado |
|------|----------|---------|--------|
| 0 - Fundamentos | Bajo | Alto | ✅ Completada |
| 1 - Scaffolding | Medio | Muy alto | ✅ Completada |
| 2 - CLI Operativo | Medio | Alto | ✅ Completada |
| 3 - Introspección | Alto | Muy alto | ✅ Completada |
| 4 - Enterprise | Alto | Estratégico | ✅ Completada |

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
**Estado**: ✅ COMPLETADO (2026-01-16)

## Commits Relacionados

| Fase | Commit | Descripción |
|------|--------|-------------|
| 0 | `b4c008e` | Examples reorganization, error handling |
| 1 | `b4c008e` | kairos init scaffolding |
| 2 | `b4c008e` | kairos run, validate commands |
| 3 | `b4c008e` | kairos explain, graph, adapters |
| 4 | `7014742` | MCP Connection Pool |
| 4 | `7d22404` | Config Layering |
| 4 | `374caac` | Corporate Templates |
