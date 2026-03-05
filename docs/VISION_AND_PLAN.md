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

| Fase | Duración estimada | Objetivo principal | Entregables clave |
|------|-------------------|--------------------|-------------------|
| **0 – Preparación** | 1 sprint | Auditoría, normas y gestión | Informe de auditoría, normas de lint/commit, tablero de proyecto | 
| **1 – Documentación** | 3 sprints | Guías completas y ejemplos | Arquitectura, referencia API, tutoriales paso‑a‑paso, changelog, ejemplos 0‑3‑5‑7 | 
| **2 – Calidad de código & pruebas** | 4 sprints | Código limpio, cobertura ≥ 80 % | TODOs eliminados, lint 0, tests unit/integration/E2E, benchmarks | 
| **3 – CI/CD** | 2 sprints | Pipelines automáticos y artefactos | Build, lint, test, benchmark, Docker image, releases automáticas | 
| **4 – Consistencia de API** | 2 sprints | API ergonomía y versiones | Firmas uniformes, deprecaciones controladas, SDK opcional | 
| **5 – Rendimiento y producción** | 2 sprints | Observabilidad, load‑testing y seguridad | Profiling, alertas OTEL, pruebas de carga, escaneo de vulnerabilidades | 

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
1. **Crear el tablero de proyecto** con las épicas descritas.
2. **Abrir la primera issue** para la fase 0 (auditoría y normas).
3. **Añadir este documento** al repositorio (ya creado).
4. **Comunicar a los contributors** la nueva estructura y proceso.

Con este plan, Kairos avanzará de un proyecto **funcional** a una **plataforma de producción** robusta, bien documentada y con una experiencia de desarrollo / despliegue consistente.
