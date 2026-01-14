# A2A (agent-to-agent)

A2A define un protocolo abierto para que agentes se descubran, negocien
capacidades y colaboren sin depender de sus detalles internos.

En Kairos, A2A es un estándar de primera clase: un agente puede actuar como
runtime local, servicio remoto y herramienta para otros agentes.

## Qué resuelve

A2A permite que agentes de distintos frameworks se entiendan sin tener que
compartir implementación. El protocolo cubre descubrimiento de capacidades,
delegación de tareas con trazabilidad y un modelo común para mensajes y tareas
que encaja bien en entornos corporativos con requisitos de seguridad y
observabilidad.

## En Kairos

Kairos implementa A2A con bindings HTTP+JSON y JSON-RPC. Además, expone Agent
Cards en `/.well-known/agent-card.json` y soporta flujos de aprobación (HITL)
cuando una política requiere intervención.

## Documentos relacionados

Si quieres ampliar, revisa: [Qué es A2A](topics/what-is-a2a.md), [Conceptos clave](topics/key-concepts.md),
[Listo para empresa](topics/enterprise-ready.md), [Descubrimiento de agentes](topics/agent-discovery.md),
[Bindings HTTP+JSON / JSON-RPC](topics/bindings.md) y [A2A y MCP](topics/a2a-and-mcp.md).

## Especificación oficial

La especificación completa está en el sitio oficial de A2A:
https://a2a-protocol.org/latest/specification
