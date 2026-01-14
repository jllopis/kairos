# Guia de clasificacion y publicación de documentacion

Este documento define como generar y publicar la documentacion de Kairos,
incluyendo tono, estructura, uso de listas y el control de documentos que se
van procesando. La idea es que `docs/` sea la fuente unica para el site y que
los documentos que ya se hayan procesado, pero no deban publicarse, se muevan a
`docs/legacy/` o `docs/internal/`.

## Objetivo

Evitar duplicidades, reducir ruido y mantener una documentacion publica clara,
con trazabilidad sobre lo que se ha extraído de los documentos internos.

## Tono y estilo

La documentacion publica debe ser directa y clara, evitando un tono
excesivamente formal o corporativo.

Recomendaciones:
- Prefiere parrafos claros a listas largas.
- Usa listas solo cuando ayuden a escanear (principios, pasos, comparativas).
- Evita numeracion tipo informe en encabezados.
- Mantiene un unico idioma: castellano.
- Evita "recuerdos" internos, instrucciones sueltas o notas operativas.

## Estructura y publicación

La web publica solo debe incluir contenido pensado para lectores externos.
El material de soporte se conserva en carpetas separadas dentro de `docs/`.

Carpetas:
- `docs/`: documentacion publica.
- `docs/legacy/`: documentos ya explotados para el site, pero que se conservan
  por historico o contexto.
- `docs/internal/`: documentos internos que no se publicarán.

## Flujo de trabajo para procesar documentos

1. Evaluar el documento y decidir si es publico, extraible o interno.
2. Extraer el contenido util y reescribirlo en la seccion publica correcta.
3. Marcar el origen como procesado.
4. Mover el documento original a `docs/legacy/` o `docs/internal/`.

## Control de documentos

Estado propuesto para el inventario:
- pendiente: no procesado.
- en revision: en proceso de reescritura.
- extraído: contenido movido a docs publicos.
- interno: no se publicará.

Inventario inicial (se actualiza según avance el trabajo):

| Documento | Estado | Destino |
| --- | --- | --- |
| docs/legacy/elevator-pitch.md | extraído | legacy |
| docs/legacy/walkthrough-a2a-httpjson-jsonrpc.md | extraído | legacy |
| docs/legacy/walkthrough-agent-discovery.md | extraído | legacy |
| docs/legacy/walkthrough-demo-a2a-agents.md | extraído | legacy (base de `docs/Demo_Kairos.md`) |
| docs/legacy/walkthrough-demo-improvements.md | extraído | legacy |
| docs/legacy/walkthrough-enhanced-agent-loop-ReAct.md | extraído | legacy |
| docs/legacy/walkthrough-explicit-planner.md | extraído | legacy |
| docs/legacy/walkthrough-governance-agentsmd.md | extraído | legacy |
| docs/legacy/walkthrough-mcp-implementation.md | extraído | legacy |
| docs/legacy/walkthrough-memory-system-implementation.md | extraído | legacy |
| docs/legacy/a2a-implementation-plan.md | extraído | legacy |
| docs/internal/adr/ | interno | internal |
| docs/internal/UI_SKELETON.md | interno | internal |
| docs/internal/NARRATIVE_GUIDE.md | interno | internal |
| docs/internal/USER_STORIES.md | interno | internal |
| docs/internal/demo-settings.json | interno | internal |
| Características Ideales de un Nuevo Framework de Agentes IA.docx | extraído | fuente externa |

Notas:
- Este inventario debe actualizarse cuando se extraiga información o se mueva
  un documento.
- La publicación se controla desde `docs-site/mkdocs.yml` con `exclude_docs`.
