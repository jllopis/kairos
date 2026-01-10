# AGENTS.md

## RESUMEN DEL PROYECTO

Cometido: Definir el framework de agentes IA "Kairos" y su vision tecnica.
Descripcion: Especificacion funcional y principios de arquitectura para un framework Go-native interoperable y observable.
Usuarios/ambito: Equipo de diseno/arquitectura y futuros contribuidores del framework.

## ESTRUCTURA DEL PROYECTO

- Raiz: especificacion y Go module.
- Codigo: `pkg/`, `examples/`.
- Tests: `pkg/**` con `*_test.go`.
- Docs: `docs/`, `EspecificacionFuncional.md`, `Caracteristicas Ideales de un Nuevo Framework de Agentes IA.docx`.
- Config/scripts: TODO: no hay.

## STACK Y HERRAMIENTAS

- Lenguajes: Go, Markdown.
- Frameworks: TODO.
- Build: `go build ./...`.
- Test: `go test ./...`.
- Lint/format: TODO.

## FLUJOS DE TRABAJO

- Ejecutar: `go run ./examples/hello-agent`.
- Testear: `go test ./...`.
- Build/Deploy: TODO.

## CONVENCIONES Y NORMAS

- Estilo de codigo: TODO.
- Naming: TODO.
- Commits/PRs: nuevas funcionalidades en rama, validar, merge a master y borrar rama provisional.

## USO DEL AGENT

- Que hacer: leer primero `EspecificacionFuncional.md` y mantener consistencia con la vision.
- Que evitar: inventar componentes, comandos o tooling no descritos.
- Como proponer cambios: sugerir updates pequeños y alineados con la arquitectura; marcar TODOs si falta informacion.

## OPTIMIZACION Y REUSO

- Reusar estructuras y APIs existentes antes de crear nuevas.
- Medir antes de optimizar; distinguir CPU, IO y GC.
- Preferir cambios backward-friendly (campos opcionales, compatibilidad).
- Usar benchmarks y pprof cuando haya hot paths.
- Evitar goroutines fugadas; propagar `context.Context`.
- Minimizar allocations y reflection fuera de serializacion controlada.
