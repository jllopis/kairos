# Kairos

Kairos es un framework de agentes IA en Go, interoperable y observable por
diseño. Está pensado para entornos reales: multiagente, gobernanza, estándares
abiertos y una base sólida para producción.

## ✨ Características principales

- **Go-native**: Alto rendimiento, tipado fuerte, despliegue sencillo
- **Interoperable**: Soporte para protocolos A2A y MCP
- **Observable**: Métricas OTEL, trazas y logs integrados
- **Production-ready**: Manejo de errores, retry policies, circuit breakers
- **Developer Experience**: CLI completo, scaffolding, config layering

## 🚀 Por dónde empezar

| Si quieres...                       | Ve a...                                           |
|-------------------------------------|---------------------------------------------------|
| Crear tu primer proyecto            | `kairos init -module github.com/tu/agente mi-agente` |
| Ejecutar tu primer agente           | [Inicio rápido](Inicio_Rapido.md)                 |
| Entender la visión del proyecto     | [Especificación Funcional](EspecificaciónFuncional.md) |
| Ver la arquitectura general         | [Arquitectura](ARCHITECTURE.md)                   |
| Aprender sobre los protocolos       | [MCP](protocols/MCP.md) / [A2A](protocols/A2A/Overview.md) |
| Ver un flujo multiagente completo   | [Demo Kairos](Demo_Kairos.md)                     |

## 🛠️ CLI y Herramientas

| Comando                              | Descripción                                    |
|--------------------------------------|------------------------------------------------|
| `kairos init`                        | Genera proyecto con scaffolding                |
| `kairos run`                         | Ejecuta agente (interactivo o con prompt)      |
| `kairos validate`                    | Valida configuración y dependencias            |
| `kairos explain`                     | Muestra arquitectura del agente                |
| `kairos status`                      | Estado del runtime y agentes                   |

Ver [CLI completo](CLI.md) para todos los comandos.

## 📋 Operaciones

| Guía                                          | Descripción                           |
|-----------------------------------------------|---------------------------------------|
| [Configuración](CONFIGURATION.md)             | Config layering, perfiles dev/prod    |
| [Manejo de errores](ERROR_HANDLING.md)        | Errores tipados, retry y recuperación |
| [Observabilidad](OBSERVABILITY.md)            | Métricas, dashboards y alertas        |
| [Guardrails de Seguridad](GUARDRAILS.md)      | Prompt injection, PII filtering       |
| [Testing Framework](TESTING.md)               | Escenarios, mocks, assertions         |
| [Templates Corporativos](CORPORATE_TEMPLATES.md) | CI/CD, Docker, observabilidad      |

## 📦 Instalación

```bash
go get github.com/jllopis/kairos
```

## 📚 Ejemplo rápido

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

## 🗂️ Ejemplos

El directorio `examples/` contiene 13 ejemplos progresivos:

| Ejemplo | Qué aprenderás |
|---------|----------------|
| `01-hello-agent` | Agente mínimo |
| `02-basic-agent` | Configuración básica |
| `03-memory-agent` | Memoria semántica |
| `04-skills-agent` | SKILLs locales |
| `05-mcp-agent` | Tools via MCP |
| `06-explicit-planner` | Planner DAG |
| `07-multi-agent-mcp` | Multi-agente |
| `08-governance-policies` | Governance |
| `09-error-handling` | Manejo de errores |
| `10-resilience-patterns` | Retry, circuit breaker |
| `11-observability` | OTEL, métricas |
| `12-production-layout` | Estructura enterprise |
| `13-mcp-pool` | Pool de conexiones MCP |
| `14-guardrails` | Seguridad: prompt injection, PII |
| `15-testing` | Testing framework |

---

*Para el roadmap completo, ver [ROADMAP.md](ROADMAP.md).*
