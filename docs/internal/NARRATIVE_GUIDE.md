# Narrative Guide: Kairos Demo

Este documento resume el mensaje clave para presentar Kairos a developers
que vienen de frameworks Python (LangChain / AutoGen / CrewAI), sin romper
el ADN A2A-native ni la adhesion a estandares.

## Lo que es

- Framework Go-native para agentes IA con procesos reales y transporte A2A.
- Interoperable por diseno: agentes en distintos runtimes/hosts hablan el
  mismo contrato.
- Observabilidad de primera clase: streaming semantico estable y trazas
  listas para UI/CLI.
- Modelo mental explicito: roles, tasks y eventos como conceptos visibles.

## Lo que NO es

- No es in-process ni una simulacion de conversacion en memoria.
- No es un framework Python ni dependiente de un runtime unico.
- No es un wrapper de prompts: la comunicacion es un contrato A2A real.
- No toca el proto A2A ni los stores en esta fase.

## Modelo mental claro (sin sorpresas)

- Role manifest: describe responsabilidad, inputs/outputs y limites.
- Task core: IDs, estado y resultado trazables a traves del flujo.
- Event taxonomy: tipos estables para streaming semantico y logging.

## Por que importa

- Los devs de CrewAI/AutoGen entienden roles + tasks + eventos al instante.
- Los operadores pueden observar y auditar sin custom logs ad-hoc.
- La demo no es marketing: expone APIs reales de libreria.

## Como contarlo en una frase

"Se parece a CrewAI o AutoGen en modelo mental, pero es distribuido,
A2A-native, estandar y serio." 

## Referencias

- docs/EVENT_TAXONOMY.md
- docs/TASKS.md
- docs/legacy/walkthrough-demo-improvements.md
