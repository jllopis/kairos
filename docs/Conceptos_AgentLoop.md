# Agent loop (ReAct)

Kairos soporta un loop ReAct: el agente razona, ejecuta herramientas y
construye una respuesta final en varios turnos.

## Flujo base

El ciclo combina pensamiento, acción y observación antes de responder.

Este flujo permite resolver tareas que requieren acciones externas antes de
responder.

Ejemplo sencillo:

```
Usuario: "Cuanto es 10 + 5?"
Agente: "Necesito calcularlo."
Acción: tool Calculadora con "10 + 5"
Observación: "15"
Respuesta final: "15"
```
