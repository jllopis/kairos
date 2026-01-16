# Kairos Orchestrator Platform - Design Document

## Resumen Ejecutivo

Este documento analiza cómo construir una plataforma de orquestación/coordinación para Kairos que permita:
- Ejecutar flujos de trabajo (workflows)
- Ejecutar agentes individuales
- Interacciones directas con LLMs
- Control visual estilo n8n/Temporal

La recomendación principal es: **mantener Kairos como framework/biblioteca y construir el orquestador como proceso separado** que consume las APIs de Kairos.

## Decisión: Dos Repositorios

| Repositorio | Tipo | Descripción |
|-------------|------|-------------|
| `kairos` | Biblioteca/Framework | Core del framework: runtime, A2A, MCP, planner, governance, LLM providers |
| `kairosctl` | Aplicación/Orquestador | Herramienta de orquestación: scheduling, workflows persistentes, registry, UI, control plane |

`kairosctl` importa `kairos` como dependencia:
```go
import (
    "github.com/jllopis/kairos/pkg/agent"
    "github.com/jllopis/kairos/pkg/a2a"
    "github.com/jllopis/kairos/pkg/planner"
    "github.com/jllopis/kairos/pkg/llm"
)
```

## Análisis de Opciones

### Opción A: Orquestador embebido en Kairos
❌ **No recomendado**

**Problemas:**
- Mezcla de responsabilidades: framework vs aplicación
- Acoplamiento de ciclos de release
- Complejidad innecesaria para usuarios que solo quieren la biblioteca
- Dificultad para escalar el control plane independientemente

### Opción B: Orquestador como proceso separado (Recomendado)
✅ **Recomendado**

```
┌─────────────────────────────────────────────────────────────┐
│                    Kairos Orchestrator                       │
│  ┌─────────────────────────────────────────────────────┐    │
│  │              Control Plane / API                     │    │
│  │  - REST/gRPC API para gestión                       │    │
│  │  - WebSocket para streaming de eventos              │    │
│  │  - Scheduler/Queue para ejecuciones                 │    │
│  └─────────────────────────────────────────────────────┘    │
│  ┌─────────────────────────────────────────────────────┐    │
│  │                 Web UI (HTMX/React)                  │    │
│  │  - Editor visual de workflows                       │    │
│  │  - Vista de ejecuciones/trazas                      │    │
│  │  - Panel de agentes                                 │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
                            │
                            │ A2A / gRPC / HTTP
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                    Kairos Framework                          │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐       │
│  │ Agent 1  │ │ Agent 2  │ │ Agent N  │ │ Workflow │       │
│  │ (A2A)    │ │ (A2A)    │ │ (A2A)    │ │ Executor │       │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘       │
│  ┌─────────────────────────────────────────────────────┐    │
│  │   Runtime | Planner | Memory | MCP | Governance     │    │
│  └─────────────────────────────────────────────────────┐    │
└─────────────────────────────────────────────────────────────┘
```

**Ventajas:**
- Separación clara de responsabilidades
- Kairos sigue siendo una biblioteca limpia
- El orquestador puede evolucionar independientemente
- Escala diferenciada (control plane vs workers)
- Usuarios pueden usar Kairos sin el orquestador

## Requisitos para Compatibilidad Forward

Para que Kairos soporte este modelo de orquestación sin bloqueos, necesitamos asegurar:

### 1. APIs Estables para Control Externo

| Componente | Estado Actual | Acción Necesaria |
|------------|---------------|------------------|
| A2A gRPC | ✅ Implementado | Mantener estabilidad |
| A2A HTTP+JSON | ✅ Implementado | Mantener estabilidad |
| Task lifecycle | ✅ core.Task | Mantener estabilidad |
| Agent discovery | 🔄 En progreso | Completar discovery patterns |
| Health checks | ✅ core.Health | Mantener estabilidad |
| Event streaming | ✅ EventEmitter | Mantener estabilidad |

### 2. Interfaces que el Orquestador Consumirá

```go
// El orquestador usará estas interfaces existentes de Kairos:

// 1. Para gestionar agentes remotos
a2a.Client // SendMessage, GetTask, ListTasks, CancelTask

// 2. Para ejecutar flujos localmente
planner.Executor // Execute(graph, state)

// 3. Para interacciones directas con LLM
llm.Provider // Complete(), CompleteWithTools()

// 4. Para observabilidad
// OTEL traces + core.EventEmitter

// 5. Para governance
governance.PolicyEngine // Evaluate()
```

### 3. Extensiones Futuras en Kairos (sin romper lo actual)

```go
// Futuro: pkg/orchestration/client.go
// Wrapper que facilita la integración con el orquestador

type OrchestratorClient interface {
    // Registrar agente en el orquestador
    Register(ctx context.Context, card a2a.AgentCard) error
    
    // Reportar heartbeat/health
    Heartbeat(ctx context.Context, status HealthStatus) error
    
    // Pull de tareas asignadas (alternativa a push A2A)
    PollTasks(ctx context.Context) ([]Task, error)
}
```

## Componentes del Orquestador (proceso separado)

### 1. Control Plane API

```
POST   /api/v1/workflows           # Crear workflow
GET    /api/v1/workflows           # Listar workflows
POST   /api/v1/workflows/:id/run   # Ejecutar workflow
GET    /api/v1/runs                # Listar ejecuciones
GET    /api/v1/runs/:id            # Detalle de ejecución
DELETE /api/v1/runs/:id            # Cancelar ejecución

GET    /api/v1/agents              # Listar agentes registrados
POST   /api/v1/agents/:id/invoke   # Invocar agente directamente
GET    /api/v1/agents/:id/tasks    # Tasks del agente

POST   /api/v1/llm/complete        # Interacción directa con LLM
WS     /api/v1/llm/stream          # Streaming LLM
```

### 2. Workflow Definition (compatible con Kairos planner)

```yaml
# Mismo formato que pkg/planner/graph.go
id: customer-support-workflow
name: Customer Support Flow
nodes:
  - id: classify
    type: agent
    config:
      agent_id: classifier-agent
      
  - id: route
    type: decision
    config:
      conditions:
        - "output.classify.category == 'billing'"
        - "output.classify.category == 'technical'"
        
  - id: billing-agent
    type: agent
    config:
      agent_id: billing-specialist
      
  - id: tech-agent
    type: agent
    config:
      agent_id: tech-support
      
edges:
  - from: classify
    to: route
  - from: route
    to: billing-agent
    condition: "output.classify.category == 'billing'"
  - from: route
    to: tech-agent
    condition: "output.classify.category == 'technical'"
```

### 3. Modos de Ejecución

| Modo | Descripción | Implementación |
|------|-------------|----------------|
| **Workflow** | Grafo de agentes + decisiones | `planner.Executor` + A2A |
| **Agent** | Invocación directa de un agente | `a2a.Client.SendMessage` |
| **LLM** | Chat directo con modelo | `llm.Provider.Complete` |
| **Hybrid** | Workflow con nodos LLM puros | Combinación |

### 4. Persistencia del Orquestador

```
orchestrator/
├── store/
│   ├── workflows.go     # Definiciones de workflows
│   ├── runs.go          # Ejecuciones y su estado
│   ├── agents.go        # Registry de agentes
│   └── schedules.go     # Programación de ejecuciones
```

Opciones de backend: SQLite (dev), PostgreSQL (prod)

## Principios de Diseño para No Bloquear el Desarrollo

### ✅ Hacer Ahora en Kairos

1. **Mantener interfaces estables** (`core.Agent`, `core.Task`, `llm.Provider`)
2. **Completar discovery patterns** (Config/WellKnown/Registry)
3. **No acoplar el CLI/UI actual al runtime** (ya es así)
4. **Mantener A2A como API principal de comunicación**
5. **EventEmitter para streaming de eventos** (ya existe)

### ⚠️ Evitar en Kairos

1. **NO añadir scheduling/queue al runtime** → responsabilidad del orquestador
2. **NO añadir workflow persistence al core** → el orquestador lo hace
3. **NO acoplar la UI web al runtime** → separar
4. **NO implementar registry centralizado** → el orquestador provee esto

### 📋 Interfaces a Estabilizar

```go
// Estas interfaces NO deben cambiar de forma incompatible:

// pkg/core/interfaces.go
type Agent interface {
    ID() string
    Role() string
    Skills() []Skill
    Memory() Memory
    Run(ctx context.Context, input any) (any, error)
}

// pkg/core/task.go  
type Task struct { ... }  // estructura estable

// pkg/a2a/client.go
type Client interface {
    SendMessage(ctx, taskID, message) error
    GetTask(ctx, taskID) (*Task, error)
    ListTasks(ctx, filter) ([]*Task, error)
    CancelTask(ctx, taskID) error
}

// pkg/llm/provider.go
type Provider interface {
    Complete(ctx, messages) (string, error)
    CompleteWithTools(ctx, messages, tools) (Response, error)
}
```

## Roadmap Propuesto

### Fase Actual (M7) - Sin cambios
Completar governance y aprovals tal como está planificado.

### Fase 8 (CLI/UI) - Ajuste menor
La UI actual (`--web`) puede evolucionar hacia un "modo standalone" o convertirse en la base del orquestador. Recomendación: mantenerla ligera.

### Nueva Fase: kairosctl MVP (Post M8)

```
- [ ] Crear repo `kairosctl`
- [ ] Control plane API (REST)
- [ ] Workflow store (SQLite)
- [ ] Agent registry (pull de AgentCards vía discovery)
- [ ] Ejecutor de workflows via A2A
- [ ] UI básica (HTMX, reutilizar de kairos --web)
```

### Nueva Fase: kairosctl Avanzado

```
- [ ] Scheduler (cron-like)
- [ ] Queue distribuida (opcional: NATS, Redis)
- [ ] Multi-tenancy
- [ ] Integración con LLM directa (bypass agent)
- [ ] Editor visual de workflows
```

## Ejemplo: Flujo de Ejecución

```
Usuario                 Orchestrator              Kairos Agents
   │                         │                          │
   │  POST /workflows/run    │                          │
   │ ───────────────────────>│                          │
   │                         │                          │
   │                         │  A2A SendMessage         │
   │                         │ ────────────────────────>│ Agent 1
   │                         │                          │
   │                         │<──────── Response ───────│
   │                         │                          │
   │                         │  Evaluate condition      │
   │                         │  (workflow logic)        │
   │                         │                          │
   │                         │  A2A SendMessage         │
   │                         │ ────────────────────────>│ Agent 2
   │                         │                          │
   │                         │<──────── Response ───────│
   │                         │                          │
   │<── Run completed ───────│                          │
   │                         │                          │
```

## Conclusiones

1. **El orquestador debe ser un proceso separado** - Kairos es framework, el orquestador es aplicación.

2. **Kairos ya tiene las piezas necesarias**:
   - A2A para comunicación entre agentes
   - Planner para ejecutar grafos
   - Task lifecycle para tracking
   - EventEmitter para streaming
   - OTEL para observabilidad

3. **Cambios actuales en Kairos son seguros** si mantenemos:
   - Interfaces core estables
   - A2A como protocolo de comunicación
   - Separación clara runtime/control plane

4. **El orquestador consumirá Kairos** como biblioteca para:
   - Ejecutar planner.Executor localmente
   - Comunicarse con agentes vía A2A
   - Interactuar con LLMs directamente

5. **No hay conflicto** entre el desarrollo actual de Kairos y esta visión futura.

## Referencias

- [n8n Architecture](https://docs.n8n.io/hosting/)
- [Temporal Architecture](https://docs.temporal.io/concepts)
- [Kairos A2A Bindings](../protocols/A2A/topics/bindings.md)
- [Kairos Planner](../Conceptos_Planner.md)
