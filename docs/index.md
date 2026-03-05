# Kairos

Kairos es un framework de agentes IA en Go, interoperable y observable por
diseño. Está pensado para entornos reales: multiagente, gobernanza, estándares
abiertos y una base sólida para producción.

## Características principales

- **Go-native**: Alto rendimiento, tipado fuerte, despliegue sencillo
- **Interoperable**: Soporte para protocolos A2A y MCP
- **Observable**: Métricas OTEL, trazas y logs integrados
- **Production-ready**: Manejo de errores, retry policies, circuit breakers
- **Developer Experience**: CLI completo, scaffolding, config layering

## Por dónde empezar

| Si quieres...                       | Ve a...                                           |
|-------------------------------------|---------------------------------------------------|
| Crear tu primer proyecto            | `kairos init -module github.com/tu/agente mi-agente` |
| Ejecutar tu primer agente           | [Inicio rápido](Inicio_Rapido.md)                 |
| Entender la visión del proyecto     | [Especificación Funcional](EspecificaciónFuncional.md) |
| Ver la arquitectura general         | [Arquitectura](ARCHITECTURE.md)                   |
| Consultar la referencia de API      | [API](API.md)                                     |
| Aprender sobre los protocolos       | [MCP](protocols/MCP.md) / [A2A](protocols/A2A/Overview.md) |
| Ver un flujo multiagente completo   | [Playbook](../examples/playbook/README.md)        |
| Contribuir al proyecto              | [Guía de desarrollo](DEVELOPER_GUIDE.md)          |

## Visión y planificación

| Documento | Descripción |
|-----------|-------------|
| [Visión y plan de evolución](VISION_AND_PLAN.md) | Fases del roadmap, métricas de progreso |
| [Roadmap](ROADMAP.md)                            | Roadmap detallado con hitos y prioridades |
| [Especificación Funcional](EspecificaciónFuncional.md) | Visión técnica completa del framework |

## Arquitectura y conceptos

| Documento | Descripción |
|-----------|-------------|
| [Arquitectura](ARCHITECTURE.md)              | Capas, componentes, diagramas C4 y flujos |
| [Agent Loop](Conceptos_AgentLoop.md)         | Bucle ReAct del agente |
| [Planner](Conceptos_Planner.md)              | Planificador DAG declarativo |
| [Memoria](Conceptos_Memoria.md)              | Memoria semántica y conversacional |
| [Memoria conversacional](CONVERSATION_MEMORY.md) | Gestión de contexto en conversaciones |
| [Skills](Skills.md)                          | Sistema de AgentSkills |
| [Tasks](TASKS.md)                            | API de tareas |
| [Taxonomía de eventos](EVENT_TAXONOMY.md)    | Tipos de eventos del runtime |

## CLI y herramientas

| Comando                              | Descripción                                    |
|--------------------------------------|------------------------------------------------|
| `kairos init`                        | Genera proyecto con scaffolding                |
| `kairos run`                         | Ejecuta agente (interactivo o con prompt)      |
| `kairos validate`                    | Valida configuración y dependencias            |
| `kairos explain`                     | Muestra arquitectura del agente                |
| `kairos status`                      | Estado del runtime y agentes                   |

Ver [CLI completo](CLI.md) para todos los comandos.

## Operaciones y configuración

| Guía | Descripción |
|------|-------------|
| [Configuración](CONFIGURATION.md)             | Config layering, perfiles dev/prod    |
| [LLM Providers](PROVIDERS.md)                 | OpenAI, Anthropic, Qwen, Gemini       |
| [Conectores](CONNECTORS.md)                   | OpenAPI, GraphQL, gRPC, SQL → tools automáticos |
| [Manejo de errores](ERROR_HANDLING.md)        | Errores tipados, retry y recuperación |
| [Guía de integración](INTEGRATION_GUIDE.md)   | Integración de errores y observabilidad |
| [Observabilidad](OBSERVABILITY.md)            | Métricas, dashboards y alertas        |
| [Exportación de métricas](METRICS_EXPORT.md)  | Arquitectura de exportación de métricas |
| [Guardrails de Seguridad](GUARDRAILS.md)      | Prompt injection, PII filtering       |
| [Gobernanza](governance-usage.md)             | Políticas de gobernanza y uso         |
| [Gobernanza HITL](governance-hitl.md)         | Human-in-the-loop                     |
| [Testing Framework](TESTING.md)               | Escenarios, mocks, assertions         |
| [Templates Corporativos](CORPORATE_TEMPLATES.md) | CI/CD, Docker, observabilidad      |

## Protocolos

| Protocolo | Descripción |
|-----------|-------------|
| [MCP (Model Context Protocol)](protocols/MCP.md) | Descubrimiento y ejecución de herramientas |
| [A2A (Agent-to-Agent)](protocols/A2A/Overview.md) | Comunicación entre agentes distribuidos |

## Instalación

```bash
go get github.com/jllopis/kairos
```

## Ejemplo rápido

```go
package main

import (
    "context"
    "github.com/jllopis/kairos/pkg/agent"
)

func main() {
    ag, _ := agent.New(
        agent.WithName("mi-agente"),
        agent.WithModel("gpt-4"),
    )

    result, _ := ag.Run(context.Background(), "Hola, ¿qué puedes hacer?")
    println(result)
}
```

## Ejemplos

El directorio `examples/` contiene 20 ejemplos progresivos:

| Ejemplo | Qué aprenderás |
|---------|----------------|
| [`01-hello-agent`](../examples/01-hello-agent/README.md) | Agente mínimo |
| [`02-basic-agent`](../examples/02-basic-agent/README.md) | Configuración básica |
| [`03-memory-agent`](../examples/03-memory-agent/README.md) | Memoria semántica |
| [`04-skills-agent`](../examples/04-skills-agent/README.md) | Skills locales |
| [`05-mcp-agent`](../examples/05-mcp-agent/README.md) | Tools via MCP |
| [`06-explicit-planner`](../examples/06-explicit-planner/README.md) | Planner DAG |
| [`07-multi-agent-mcp`](../examples/07-multi-agent-mcp/README.md) | Multi-agente |
| [`08-governance-policies`](../examples/08-governance-policies/README.md) | Governance |
| [`09-error-handling`](../examples/09-error-handling/README.md) | Manejo de errores |
| [`10-resilience-patterns`](../examples/10-resilience-patterns/README.md) | Retry, circuit breaker |
| [`11-observability`](../examples/11-observability/README.md) | OTEL, métricas |
| [`12-production-layout`](../examples/12-production-layout/README.md) | Estructura enterprise |
| [`13-mcp-pool`](../examples/13-mcp-pool/README.md) | Pool de conexiones MCP |
| [`14-guardrails`](../examples/14-guardrails/README.md) | Seguridad: prompt injection, PII |
| [`15-testing`](../examples/15-testing/README.md) | Testing framework |
| [`16-providers`](../examples/16-providers/README.md) | LLM providers: auth y tokens |
| [`17-openapi-connector`](../examples/17-openapi-connector/README.md) | REST API → tools automáticos |
| [`18-streaming`](../examples/18-streaming/README.md) | Respuestas en tiempo real |
| [`19-graphql-connector`](../examples/19-graphql-connector/README.md) | GraphQL → tools automáticos |
| [`20-conversation-memory`](../examples/20-conversation-memory/README.md) | Memoria conversacional |

Para un walkthrough narrativo completo, ver el [Playbook](../examples/playbook/README.md).

## Documentación interna

| Documento | Descripción |
|-----------|-------------|
| [ADRs](internal/adr/) | Decisiones de arquitectura registradas |
| [Guía AGENTS.md](agents-md.md) | Cómo crear ficheros AGENTS.md |

---

*Para el roadmap completo, ver [ROADMAP.md](ROADMAP.md).*
