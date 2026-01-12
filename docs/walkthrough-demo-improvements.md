# Walkthrough: Demo Improvements Plan

Este walkthrough recoge el plan de mejoras para que Kairos sea mas manejable
para developers que vienen de LangChain / AutoGen / CrewAI, sin romper la
arquitectura A2A-native ni la adhesion a estandares. Las mejoras deben vivir
en la libreria (core), y la demo solo mostrarlas.

## Principios y acuerdos

- No tocar el proto A2A ni los stores por ahora.
- La fuerza del framework es la adhesion a estandares para evitar sorpresas.
- Las mejoras son de experiencia y modelo mental, no de “copiar” frameworks.
- Hacer explicito lo que hoy ya es correcto pero esta implicito.
- La demo no define el API: lo muestra.

## Objetivo

Que un developer vea la demo y piense:
"Esto se parece a CrewAI / AutoGen, pero distribuido, estandar y serio."

## Fase 1: Claridad semantica en la libreria (alto impacto, bajo riesgo)

1) Role/manifest como metadatos de libreria
   - API en core para definir `role`, `responsibility`, `inputs`, `outputs`, `constraints`.
   - Puede convivir con AgentCard sin tocar proto A2A.
   - Objetivo: visibilidad inmediata del rol y limites.
   - Demo: `role.yaml` por agente que alimenta el API.

2) Event taxonomy semantica en streaming
   - Definir y emitir eventos con tipos estables desde la libreria.
   - Tipos minimos:
     - `agent.thinking`
     - `agent.task.started`
     - `agent.task.completed`
     - `agent.delegation`
     - `agent.error`
   - Campos minimos: `type`, `agent`, `task_id`, `timestamp`, `payload`.
   - Objetivo: logs y streaming “legibles” y UI-friendly.
   - Demo: visualizar y formatear estos eventos.
   - Ver `docs/EVENT_TAXONOMY.md`.

3) Narrativa y docs
   - Aclarar “lo que NO es / lo que SI es”.
   - Comparativa directa con frameworks Python.
   - Implementado en `docs/NARRATIVE_GUIDE.md`.

## Fase 2: Task como entidad de primer nivel (libreria)

4) Task en core (API estable)
   - Estructura tipo `Task` con `ID`, `Goal`, `AssignedTo`, `Status`, `Result`.
   - Mapeo a A2A Task/Message sin modificar proto/stores.
   - Trazabilidad: `task_id` y `task_goal` en logs/traces/eventos.
   - Demo: crear tasks explicitas para mostrar el flujo.

## Fase 3: UX de demo (experiencia de arranque)

5) Builder fluido (Go) para demo
   - Facade tipo:
     ```
     kairos.NewSystem().
       WithAgent(...).
       WithFlow(...).
       Run()
     ```
   - Solo en demo: no altera runtime core.
   - Revisitaremos moverlo a core cuando Task/Role/Event esten fijados.
   - Implementado en `demoKairos/internal/demo/system.go` y usado desde `demoKairos/cmd/demo`.

6) Script/entrypoint unificado
   - Un comando para levantar demo con defaults.
   - Logs formateados para seguir el flujo.
   - Implementado en `demoKairos/cmd/demo` y expuesto via `demoKairos/scripts/run-demo.sh`.

## Entregables esperados

- API de Role/manifest en core + `demoKairos/docs/role-*.yaml`.
- Event taxonomy documentada y emitida desde la libreria.
- Task como entidad en core (sin cambios en proto/stores).
- Builder fluido y entrypoint de demo.

## Criterios de aceptacion

- Un developer entiende roles y flujo sin leer codigo.
- Los logs/streaming muestran eventos semanticamente claros.
- La demo se arranca con un comando y produce un flujo legible.
