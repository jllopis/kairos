# Demo Kairos

La demo de Kairos muestra un flujo multiagente real con planner, A2A y MCP. El
objetivo es enseñar el modelo mental del framework sin esconder detalles
importantes.

## Componentes principales

La demo vive en `demoKairos/` y se apoya en agentes especializados y un
orquestador con planner explícito. Cada agente es un proceso A2A que puede
exponer herramientas vía MCP.

Incluye:

- Orquestador con planner y streaming A2A.
- Agente de conocimiento con RAG vía MCP.
- Agente de hojas de cálculo con tools MCP.
- Cliente que consume el stream de eventos.

## Flujo de ejecución

1. El cliente llama a `SendStreamingMessage` en el orquestador.
2. El orquestador ejecuta el grafo del planner y emite eventos semánticos.
3. Se delega en los agentes de conocimiento y hojas de cálculo.
4. El orquestador sintetiza la respuesta y la devuelve por streaming.

## Planner del orquestador

El grafo del orquestador define pasos como detección de intención,
consulta de conocimiento, consulta de hojas y síntesis final. El objetivo es
mantener el flujo determinista y trazable.

## MCP en la demo

Cada agente expone herramientas via MCP y el orquestador invoca esas tools para
resolver partes del flujo. Esto permite separar capacidad semántica (skill) de
implementación concreta (tool).

## Eventos y trazas

La demo emite eventos semánticos (`agent.task.started`, `agent.delegation`,
`agent.task.completed`, etc.) y propaga trazas entre agentes para observabilidad
end-to-end.

## Roles y tareas

Los agentes cargan manifiestos de rol desde `demoKairos/docs/role-*.yaml` y el
orquestador crea una entidad Task en core para mantener contexto y trazabilidad
sin tocar el proto A2A.
