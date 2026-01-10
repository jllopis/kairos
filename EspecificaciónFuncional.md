# Especificación del Framework de Agentes IA `Kairos`

## 1. Visión y posicionamiento

El framework será un **framework de agentes IA de propósito general**, con **foco prioritario en agentes corporativos y de desarrollo**, pero **sin excluir** otros dominios (asistentes, automatización, data, DevOps, etc.).

Se posiciona como:

> **“El framework de referencia para agentes IA en Go, interoperable, observable, distribuido y gobernable desde el diseño.”**

Sus pilares diferenciales serán:

* **Implementación en Go** (core + SDK)
* **Interoperabilidad nativa** (MCP, A2A, AGENTS.md)
* **Observabilidad estándar con OpenTelemetry**
* **Arquitectura multiagente distribuida**
* **Control técnico + herramientas visuales**
* **Pensado para producción desde el día 1**

---

## 2. Lenguaje y SDK: Go como decisión estratégica

### 2.1 Core del framework en Go

El **core del framework se implementa en Go**, no como accidente, sino como decisión estratégica:

**Justificación técnica:**

* Vacío real en el ecosistema de agentes IA en Go

* Excelente para:

  * concurrencia (goroutines, channels)
  * agentes distribuidos
  * workloads de larga duración
  * binarios autosuficientes (edge / on-prem / cloud)

* Encaja perfectamente con:

  * Kubernetes
  * observabilidad
  * infra corporativa
  * tooling DevOps

El framework **no depende de Python para funcionar**.

---

### 2.2 SDK inicial en Go (first-class citizen)

El **SDK oficial inicial será Go**, con:

* API tipada
* composición explícita
* cero magia implícita
* control fino del ciclo de vida del agente

Ejemplo conceptual:

```go
agent := agent.New(
  agent.WithRole("IncidentAnalyzer"),
  agent.WithPlanner(planner.Graph),
  agent.WithMemory(memory.Vector),
  agent.WithTools(mcp.Tools()),
)
```

Otros SDKs (Python / TS) **podrán existir después**, pero:

* el core **no dependerá de ellos**
* no habrá “core en Python + wrapper en Go”

---

## 3. Arquitectura general

### 3.1 Arquitectura modular por capas

```
┌──────────────────────────────┐
│        Interfaces UI         │  (Web / CLI)
├──────────────────────────────┤
│      Agent Control Plane     │  (API / Auth / Policies)
├──────────────────────────────┤
│   Multi-Agent Runtime Core   │  (Go)
├──────────────────────────────┤
│ Planner | Memory | Tools     │
├──────────────────────────────┤
│ MCP | A2A | AGENTS.md        │
├──────────────────────────────┤
│ OpenTelemetry | Storage      │
└──────────────────────────────┘
```

---

## 4. Observabilidad: OpenTelemetry como estándar obligatorio

### 4.1 Decisión explícita

> **Toda la observabilidad del framework se implementa con OpenTelemetry (OTel).**

No es opcional. No es “plugin”.

Incluye:

* traces distribuidos
* métricas
* logs estructurados

Y, lo más importante:

* continuidad de trace entre agentes (A2A)
* visibilidad real de sistemas multiagente

---

### 4.2 Qué se instrumenta

Cada agente y cada ejecución genera:

* **Traces**

  * ejecución del agente
  * pasos del planner
  * llamadas a herramientas
  * llamadas a otros agentes (A2A)
* **Metrics**

  * latencia por paso
  * tokens usados
  * errores por agente
  * retries
* **Logs estructurados**

  * razonamiento
  * decisiones
  * resultados intermedios

Todo con **context propagation estándar**.

Esto permite integración directa con:

* Grafana
* Prometheus
* Jaeger
* Tempo
* Datadog
* New Relic
* etc.

---

### 4.3 Observabilidad multiagente

Se soporta **trazado distribuido entre agentes**:

* un agente A llama a agente B
* el trace continúa
* el grafo completo queda visible

Esto **resuelve uno de los mayores vacíos actuales** en frameworks de agentes.

---

## 5. Interoperabilidad nativa (no “adaptadores”)

### 5.1 MCP (Model Context Protocol)

El framework:

* Implementa **cliente y servidor MCP**
* Puede:

  * consumir herramientas MCP
  * exponer agentes como herramientas MCP

**Resultado:**

* Cualquier agente puede usar:

  * herramientas externas
  * bases de conocimiento
  * sistemas corporativos
* Sin SDK específico por integración

---

### 5.2 A2A / ACP (Agent-to-Agent)

El framework implementa **A2A / ACP como protocolo nativo**:

* agentes descubren otros agentes
* comunicación remota
* handoff de tareas
* federación entre organizaciones

Un agente es:

* **un runtime local**
* **un servicio A2A**
* **una herramienta MCP**

Todo al mismo tiempo.

---

### 5.3 AGENTS.md (soporte automático)

En el arranque, el framework:

  * busca `AGENTS.md` en el entorno
  * lo parsea
  * lo incorpora al contexto base del agente

**Uso automático**, sin configuración explícita.

AGENTS.md define:

* reglas de comportamiento
* convenciones del repositorio
* límites del agente
* estándares de código / seguridad

Esto habilita:

* agentes de desarrollo
* agentes CI/CD
* agentes de revisión
* agentes infra

### 5.4 Modelo de Tools / Skills (estándares first)

#### 5.4.1 Principio rector

> KAIROS implementa las herramientas de los agentes exclusivamente sobre estándares abiertos y consolidados.

En concreto, KAIROS soporta de forma nativa:

* Skills → estándar AgentSkills (agentskills.io)
* Tools → Model Context Protocol (MCP)
* Agentes → A2A / ACP (Agent-to-Agent)

No existen APIs propietarias para tool-calling.
No existen “helpers mágicos” fuera de estos estándares.

---

#### 5.4.2 Tres niveles de herramientas (compatibles entre sí)

El framework soporta **tres tipos de herramientas**, todas **nativas**, todas **interoperables**:

##### 1) Skills (primer nivel, semántico)

En KAIROS, Skills son exactamente el estándar definido en agentskills.io:

Consultar la Especificación en https://agentskills.io/specification

No es una reinterpretación.
No es una abstracción propia.

KAIROS:

* lee
* interpreta
* expone
* ejecuta

Skills tal y como define el estándar AgentSkills.

Un agente **razona en términos de skills**, no de funciones.

---

#####  2) MCP Tools (implementación estándar)

Cada Skill puede estar respaldado por **una o varias implementaciones MCP**.

KAIROS implementa:

* cliente MCP
* servidor MCP

Esto permite:

* consumir tools externas
* exponer skills propias como tools
* federar tooling corporativo

KAIROS no define ningún formato alternativo.

---

#####  3) Comunicación entre Agentes (A2A)

KAIROS implementa A2A como:

* protocolo de comunicación
* sistema de descubrimiento
* mecanismo de delegación

Un agente puede ser **invocado como herramienta** por otro agente:

* vía **A2A**
* con contrato explícito
* con trazado distribuido

Esto habilita:

* delegación real
* especialización
* arquitecturas de agentes en red

---

#### 5.4.3 Resumen del modelo de tools

| Nivel | Qué es                  | Estándar |
| ----- | ----------------------- | -------- |
| Skill | Capacidad semántica     | AgentSkills   |
| Tool  | Implementación concreta | MCP      |
| Agent | Tool inteligente        | A2A      |

**Esto rellena un vacío real**:
la separación clara entre *qué* puede hacerse (Skill) y *cómo* se hace (Tool).
Esta separación no existe clara en la mayoría de frameworks actuales.
En KAIROS es fundacional.

**Binding Skill → Tool**

Un Skill puede utilizar herramientas proporcionadas por MCPs. Se resuelve dinámicamente contra una o varias tools MCP.

El agente puede decidir, en base a los Skills:

* qué tool usar
* en qué orden
* con qué fallback

Esto complementa el uso de la descripción proporcionada por los MCP.

---

## 6. Planificación (Planner)

### 6.1 Planner explícito + emergente

El framework soporta **dos modelos simultáneos**:

1. **Planner explícito**

   * grafos dirigidos
   * nodos = agentes / herramientas / decisiones
   * determinismo
2. **Planner emergente**

   * agentes deciden siguiente acción
   * delegación dinámica
   * estilo AutoGen / CrewAI

Ambos pueden **combinarse**.

---

### 6.2 Definición por código + por ficheros

* Go (API tipada)
* YAML / JSON (declarativo)
* UI gráfica (ver sección 8)

Todo representa **el mismo modelo interno**.

---

## 7. Memoria y contexto

### 7.1 Memoria de múltiples niveles

* memoria corta (session)
* memoria larga (vectorial)
* memoria compartida entre agentes
* memoria persistente entre ejecuciones

Abstracción clara:

```go
type Memory interface {
  Store(ctx, data)
  Retrieve(ctx, query)
}
```

---

## 8. Herramientas visuales (sin traicionar lo técnico)

### 8.1 UI web de observación y control

Incluye:

* vista de agentes activos
* flujos en ejecución
* trazas OTel
* estado de memoria
* intervención manual

---

### 8.2 Editor visual de grafos

* definir flujos complejos
* exportar a YAML / Go
* versionable
* auditable

No sustituye el código:
**lo complementa**.

---

## 9. Seguridad y gobierno corporativo

Incluye de serie:

* políticas por agente
* scopes de herramientas
* human-in-the-loop
* auditoría completa
* control de costes (tokens, llamadas)

Todo alineado con:

* entornos ISO / ENS
* RGPD
* Zero Trust

---

## 10. Despliegue y operación

* binario único Go
* Docker / Kubernetes
* on-prem / cloud / edge
* escalado horizontal
* tolerancia a fallos

Agentes como:

* procesos
* pods
* servicios A2A

---

## 11. Qué huecos reales rellena este framework

| Hueco actual                            | Cómo se resuelve                   |
| --------------------------------------- | ---------------------------------- |
| No hay framework serio de agentes en Go | **Core + SDK en Go**               |
| Observabilidad ad-hoc                   | **OpenTelemetry nativo**           |
| Interoperabilidad fragmentada           | **MCP + A2A first-class**          |
| AGENTS.md ignorado                      | **Carga automática**               |
| Multiagente difícil de depurar          | **Tracing distribuido**            |
| UI vs código                            | **Modelo único, múltiples vistas** |
| Producción = bricolaje                  | **Diseñado para prod**             |

---

## 12. Resumen ejecutivo

Este framework sería:

* **Go-native**
* **interoperable por diseño**
* **observable por defecto**
* **multiagente real**
* **usable en producción**
* **alineado con estándares emergentes**
