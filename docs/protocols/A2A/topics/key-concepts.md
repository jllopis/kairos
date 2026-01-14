# Conceptos clave

A2A se apoya en conceptos básicos que se mantienen estables entre bindings.

## Agent Card

Documento JSON que describe capacidades, endpoints y requisitos de seguridad.

## Task

Unidad de trabajo con estado y ciclo de vida. Puede producir mensajes y
artefactos conforme progresa.

## Message y Parts

Un mensaje tiene un rol y uno o varios parts (texto, archivos, datos). Los
parts permiten contenido estructurado sin acoplar la implementación.

## Streaming y Push

A2A soporta actualizaciones en tiempo real vía streaming y notificaciones
asíncronas vía webhooks.
