# Visión y Plan de Evolución de Kairos

## 🎯 Visión general

Kairos es un **framework de agentes IA nativo en Go** que combina:

- **Bucles ReAct emergentes** y **planificadores declarativos** bajo una única interfaz.
- **MCP (Model Context Protocol)** para descubrimiento y ejecución de herramientas.
- **A2A (Agent‑to‑Agent)** que permite arquitecturas distribuidas y multi‑tenant.
- **Observabilidad completa** (trazas, métricas y logs estructurados) mediante OpenTelemetry.
- **Gobernanza y Guardrails** integrados por defecto para seguridad y cumplimiento.

El objetivo es convertir a Kairos en una **plataforma de producción lista** para despliegues empresariales, con una experiencia de desarrollo fluida y un plano de control (`kairosctl`) que permita orquestar agentes, skills y workflows a gran escala.

---

## 📆 Propuesta de fases (Roadmap operativizado)

| Fase | Objetivo principal | Estado | Entregables clave |
|------|--------------------|--------|-------------------|
| **0 – Preparación** | Auditoría, normas y gestión | ✅ COMPLETA | Makefile, golangci-lint config, CI workflow, VISION_AND_PLAN.md |
| **1 – Documentación** | Guías completas y ejemplos | ✅ COMPLETA | ARCHITECTURE.md (Mermaid), API.md (19 secciones), DEVELOPER_GUIDE.md, CHANGELOG.md, index.md, conceptos expandidos, cross-refs corregidos |
| **2 – Calidad de código & pruebas** | Código limpio, cobertura ≥ 80 % | Pendiente | TODOs eliminados, lint 0, tests unit/integration/E2E, benchmarks |
| **3 – CI/CD** | Pipelines automáticos y artefactos | Pendiente | Docker image, releases automáticas, Dependabot |
| **4 – Consistencia de API** | API ergonomía y versiones | Pendiente | Firmas uniformes, deprecaciones controladas, SDK opcional |
| **5 – Rendimiento y producción** | Observabilidad, load‑testing y seguridad | Pendiente | Profiling, alertas OTEL, pruebas de carga, escaneo de vulnerabilidades |

### Detalle de cada fase

#### Fase 0 – Preparación
- **Auditoría del repositorio** (`go list`, `golint`, búsqueda de TODO/FIXME).
- **Definir normas**: *golangci‑lint*, formato `go fmt`, convención de commits (Conventional Commits).
- **Crear tablero** (GitHub Projects / Jira) con épicas y tareas.
- **Criterios de “solvencia”**: documentación 100 %, cobertura ≥ 80 %, pipelines verdes, métricas dentro de umbrales.

#### Fase 1 – Documentación
- **Visión y arquitectura** (diagramas C4, flujo ReAct‑Planner, MCP & A2A).
- **Referencia de API** (markdown generados a partir de `godoc`).
- **Guía de desarrollo** (configuración entorno, lint, pruebas, CI). 
- **Tutoriales progresivos**: `hello-agent`, `explicit-plan`, `multi‑agent`, `kairosctl‑demo`.
- **Changelog automático** (skill `changelog-gen`).

#### Fase 2 – Calidad y pruebas
- Eliminar todos los `TODO`/`FIXME` y asignar historias.
- Refactor de manejo de errores (`%w`, `errors.Is/As`).
- Configurar `golangci‑lint` (deadcode, gosec, vet, etc.).
- Cobertura unitarias → **≥ 80 %** (tabla por paquete). 
- Tests de integración con servidor MCP mock (docker‑compose). 
- End‑to‑end con ejemplos completos y validación de salida. 
- Benchmarks de rutas críticas (tool resolution, memoria). 

#### Fase 3 – CI/CD
- **Workflow**: build → lint → unit → integration → e2e → benchmark.
- Docker multi‑stage y publicación en GitHub Packages.
- GitHub Action para crear releases a partir de tags (`CHANGELOG.md`).
- Dependabot / Renovate para actualizar módulos Go. 

#### Fase 4 – Consistencia de API
- Uniformizar firmas (`ctx context.Context` primero). 
- Nomenclatura coherente (CamelCase, PascalCase). 
- Documentar convenciones en `DeveloperGuide.md`. 
- Deprecaciones controladas (wrappers 1 versión). 
- (Opcional) generar SDK externo con `go generate`.

#### Fase 5 – Rendimiento & producción
- **Profiling** (`pprof`, CPU/heap). 
- Optimizar hot‑paths (MCP tool lookup, memoria). 
- Enriquecer trazas OTEL (args/result, estado interno). 
- Alertas de latencia, error‑rate y queue depth. 
- Load‑testing (`k6`, `hey`). 
- Seguridad: escaneo `govulncheck`, pruebas de guardrails (PII, prompt injection). 
- Documentar despliegue Kubernetes (HPA, ConfigMaps, Secrets). 

---

## 📈 Seguimiento y métricas de progreso

1. **Tablero de proyecto** con épicas → historias → tareas. Cada historia lleva: descripción, criterio de aceptación, estimación (puntos), responsable.
2. **Burn‑down** semanal del sprint para visualizar progreso.
3. **Indicadores de salud** (dashboards):
   - % de cobertura de pruebas.
   - Número de fallos de lint.
   - Tiempo medio de pipeline.
   - Latencia media de llamadas MCP.
4. **Revisiones de código** obligatorias (mínimo 2 reviewers). 
5. **Demo al final de cada fase** (presentación de artefactos, métricas y lecciones aprendidas).

---

## 📂 Estructura de la documentación

```
docs/
│   VISION_AND_PLAN.md   ← **este archivo**
│   ROADMAP.md           ← Roadmap existente (referencia histórica)
│   ARCHITECTURE.md
│   API.md
│   ...
```

Se añadirá un **índice** (`docs/index.md`) que enlazará este documento y los demás recursos para que los contributors encuentren rápidamente la visión y el plan de evolución.

---

## ✅ Próximos pasos inmediatos
1. **Iniciar Fase 2 — Calidad de código & pruebas**: auditar TODOs/FIXMEs, aumentar cobertura a ≥ 80 %, añadir tests de integración y E2E.
2. **Planificar Fase 3 — CI/CD**: Docker multi-stage, releases automáticas desde tags, Dependabot.
3. **Comunicar a los contributors** la nueva estructura y proceso.

Con este plan, Kairos avanzará de un proyecto **funcional** a una **plataforma de producción** robusta, bien documentada y con una experiencia de desarrollo / despliegue consistente.
