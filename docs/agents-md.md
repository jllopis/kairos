# AGENTS.md

AGENTS.md es el archivo donde dejas claro cómo deben comportarse los agentes en
un repo. Kairos lo carga al arrancar, sin configuración extra.

## Para que sirve

Te ayuda a dejar por escrito:

- Reglas de seguridad y calidad.
- Convenciones del repo y del stack.
- Límites de lo que un agente puede o no puede hacer.

## Cómo lo usa Kairos

El runtime busca `AGENTS.md`, lo incorpora al contexto base y aplica esas reglas
durante la ejecución.

## Consejos rápidos

- Mantenlo corto y accionable.
- Separa reglas obligatorias de sugerencias.
- Revísalo cuando cambie la arquitectura.

## Relación con governance

AGENTS.md define el marco de comportamiento. Las políticas de governance aplican
esas reglas en tiempo de ejecución. Para ver ejemplos y configuración, consulta
la [Guía de governance](governance-usage.md).
