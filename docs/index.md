# Kairos

Kairos es un framework de agentes IA en Go, interoperable y observable por
diseño. Está pensado para entornos reales: multiagente, gobernanza, estándares
abiertos y una base sólida para producción.

## ✨ Características principales

- **Go-native**: Alto rendimiento, tipado fuerte, despliegue sencillo
- **Interoperable**: Soporte para protocolos A2A y MCP
- **Observable**: Métricas OTEL, trazas y logs integrados
- **Production-ready**: Manejo de errores, retry policies, circuit breakers

## 🚀 Por dónde empezar

| Si quieres...                       | Ve a...                                           |
|-------------------------------------|---------------------------------------------------|
| Ejecutar tu primer agente           | [Inicio rápido](Inicio_Rapido.md)                 |
| Entender la visión del proyecto     | [Especificación Funcional](EspecificaciónFuncional.md) |
| Ver la arquitectura general         | [Arquitectura](ARCHITECTURE.md)                   |
| Aprender sobre los protocolos       | [Protocolos A2A](protocols/A2A/Overview.md)       |
| Ver un flujo multiagente completo   | [Demo Kairos](Demo_Kairos.md)                     |

## 🛠️ Operaciones

| Guía                                          | Descripción                           |
|-----------------------------------------------|---------------------------------------|
| [Manejo de errores](ERROR_HANDLING.md)        | Errores tipados, retry y recuperación |
| [Integración con agentes](INTEGRATION_GUIDE.md) | Uso en loops de agentes             |
| [Observabilidad](OBSERVABILITY.md)            | Métricas, dashboards y alertas        |
| [Exportación de métricas](METRICS_EXPORT.md)  | Configuración OTLP y backends         |

## 📦 Instalación

```bash
go get github.com/jllopis/kairos
```

## 📚 Ejemplo básico

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

---

*Para más ejemplos, consulta el directorio `examples/` en el repositorio.*
